package core

import (
	"context"
	"fmt"
	"strconv"

	"github.com/borchero/switchboard/api/v1alpha1"
	"github.com/borchero/switchboard/core/utils"
	"github.com/jetstack/cert-manager/pkg/apis/certmanager/v1alpha2"
	cmmetav1 "github.com/jetstack/cert-manager/pkg/apis/meta/v1"
	"go.borchero.com/typewriter"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type recordReconciler struct {
	*Reconciler
	log typewriter.Logger
}

// RegisterRecordReconciler adds a new reconciliation loop to watch for changes of DNSResource.
func RegisterRecordReconciler(base *Reconciler, manager ctrl.Manager, log typewriter.Logger) error {
	reconciler := &recordReconciler{base, log.With("records")}
	return ctrl.
		NewControllerManagedBy(manager).
		For(&v1alpha1.DNSRecord{}).
		Owns(&v1alpha1.DNSZoneRecord{}).
		Owns(&v1alpha2.Certificate{}).
		Complete(reconciler)
}

func (r *recordReconciler) Reconcile(request ctrl.Request) (ctrl.Result, error) {
	var record v1alpha1.DNSRecord
	return r.doReconcile(
		request, &record, r.log,
		func(log typewriter.Logger) error { return r.update(&record, log) },
		noDelete, noEmptyDelete,
	)
}

func (r *recordReconciler) update(record *v1alpha1.DNSRecord, logger typewriter.Logger) error {
	ctx := context.Background()

	// 1) Ensure zone record update
	// 1.1) Get expected zone records
	expectedRecords, err := r.listExpectedZoneRecords(*record)
	if err != nil {
		return err
	}

	// 1.2) List owned zone records
	ownedRecords, err := r.listOwnedZoneRecords(*record)
	if err != nil {
		return err
	}

	// 1.3) Compute diff
	insertset, deletionset := utils.ZoneRecordDiff(expectedRecords, ownedRecords)

	// 1.4) Create non-existing records
	if len(insertset) > 0 {
		logger.
			WithV(typewriter.KV("count", strconv.Itoa(len(insertset)))).
			Info("Creating zone records")
	}
	for _, insert := range insertset {
		if err := r.client.Create(ctx, &insert); err != nil {
			return fmt.Errorf("Failed creating DNS zone record: %s", err)
		}
	}

	// 1.5) Delete falsely existing resources
	if len(deletionset) > 0 {
		logger.
			WithV(typewriter.KV("count", strconv.Itoa(len(deletionset)))).
			Info("Deleting zone records")
	}
	for _, delete := range deletionset {
		if err := r.client.Delete(ctx, &delete); err != nil {
			return fmt.Errorf("Failed deleting DNS zone record: %s", err)
		}
	}

	// 2) Ensure that specified TLS certificate is created
	certName := record.CertificateNamespacedName()

	// 2.1) First, list all certificates that are owned by this record and whose name is "incorrect"
	if err := r.deleteOwnedCertificatesNotMatchingName(*record, certName.Name); err != nil {
		return fmt.Errorf("Failed purging stale certificates: %s", err)
	}

	// 2.2) We now create/update the certificate when the TLS setting is specified
	if record.Spec.TLS != nil {
		// 2.2.1) First, we get the certificate by name
		var existingCertificate v1alpha2.Certificate
		found := true
		if err := r.client.Get(ctx, certName, &existingCertificate); err != nil {
			if apierrs.IsNotFound(err) {
				found = false
			} else {
				return fmt.Errorf("Failed getting existing certificate by name: %s", err)
			}
		}

		if found {
			// 2.2.2) We ensure that we are the owners
			owner := metav1.GetControllerOf(&existingCertificate)
			gvs := v1alpha1.GroupVersion.String()
			if owner == nil || owner.APIVersion != gvs || owner.Kind != "DNSRecord" {
				// 2.2.3) We do not own the certificate and cannot/should not control its contents.
				// This does mean, however, that we cannot set DNS alt names and we see this as an
				// error.
				return fmt.Errorf(
					`A certificate with the name '%s' already exists but is not owned by the
					controller. Remove the certificate to resolve this error`, certName,
				)
			}
		}

		// 2.2.4) Now, we can update the existing certificate if necessary or create a new one. In
		// any case we need a "reference" certificate from which to source the updated values.
		targetCertificate, err := r.certificate(*record)
		if err != nil {
			return fmt.Errorf("Failed generating target certificate: %s", err)
		}

		if found {
			// 2.2.5) In this case, we update
			if ok := certificateUpdateIfNeeded(&existingCertificate, targetCertificate); ok {
				if err := r.client.Update(ctx, &existingCertificate); err != nil {
					return fmt.Errorf("Failed updating existing certificate: %s", err)
				}
			}
		} else {
			// 2.2.6) In this case, we create the certificate
			if err := r.client.Create(ctx, &targetCertificate); err != nil {
				return fmt.Errorf("Failed creating new certificate: %s", err)
			}
		}
	}

	// 3) Again, we don't need any finalizers as zone records are deleted automatically upon removal
	// of its owner.
	return nil
}

func (r *recordReconciler) listOwnedZoneRecords(
	record v1alpha1.DNSRecord,
) ([]v1alpha1.DNSZoneRecord, error) {
	ctx := context.Background()

	var existingRecords v1alpha1.DNSZoneRecordList
	option := &client.ListOptions{
		Namespace:     record.Namespace,
		FieldSelector: fields.OneTermEqualSelector(indexFieldOwner, record.Name),
	}
	if err := r.client.List(ctx, &existingRecords, option); err != nil {
		return nil, fmt.Errorf("Failed listing owned zone records: %s", err)
	}

	return existingRecords.Items, nil
}

func (r *recordReconciler) listExpectedZoneRecords(
	record v1alpha1.DNSRecord,
) ([]v1alpha1.DNSZoneRecord, error) {
	zoneRecords := make([]v1alpha1.DNSZoneRecord, len(record.Spec.Zones))
	for i := range record.Spec.Zones {
		zoneRecord, err := r.getExpectedZoneRecord(record, i)
		if err != nil {
			return nil, fmt.Errorf(
				"Failed generating dns zone record for zone '%s': %s",
				record.Spec.Zones[i].Name, err,
			)
		}
		zoneRecords[i] = zoneRecord
	}
	return zoneRecords, nil
}

func (r *recordReconciler) getExpectedZoneRecord(
	record v1alpha1.DNSRecord, zoneIndex int,
) (v1alpha1.DNSZoneRecord, error) {
	ctx := context.Background()
	zoneRef := record.Spec.Zones[zoneIndex]

	// 1) Get the zone to source the template values
	var zone v1alpha1.DNSZone
	if err := r.client.Get(ctx, zoneRef.NamespacedName(), &zone); err != nil {
		return v1alpha1.DNSZoneRecord{}, fmt.Errorf("Failed to get referenced zone: %s", err)
	}

	// 2) Find values
	// 2.1) IP source (must be specified, i.e. no default)
	var ipSource v1alpha1.IPSource
	if !zoneRef.IPSource.Empty() {
		ipSource = zoneRef.IPSource
	} else if !zone.Spec.RecordTemplate.IPSource.Empty() {
		ipSource = zone.Spec.RecordTemplate.IPSource
	} else {
		return v1alpha1.DNSZoneRecord{}, fmt.Errorf("Neither record nor zone provide an IP source")
	}

	if ipSource.Service != nil {
		if ipSource.Service.Namespace == "" {
			ipSource.Service.Namespace = record.Namespace
		}
		if ipSource.Service.Type == v1alpha1.ServiceIPType("") {
			ipSource.Service.Type = v1alpha1.ServiceIPTypeCluster
		}
	}

	if ipSource.Node != nil {
		if ipSource.Node.Type == v1alpha1.NodeIPType("") {
			ipSource.Node.Type = v1alpha1.NodeIPTypeExternal
		}
	}

	// 2.2) Time to live (defaults to 300 seconds)
	ttl := v1alpha1.TimeToLive(300)
	if zoneRef.TTL != nil {
		ttl = *zoneRef.TTL
	} else if record.Spec.TTL != nil {
		ttl = *record.Spec.TTL
	} else if zone.Spec.RecordTemplate.TTL != nil {
		ttl = *zone.Spec.RecordTemplate.TTL
	}

	// 3) Create the zone record
	zoneRecord := v1alpha1.DNSZoneRecord{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: fmt.Sprintf("%s-", record.Name),
			Namespace:    record.Namespace,
		},
		Spec: v1alpha1.DNSZoneRecordSpec{
			ZoneName:       zone.Name,
			DNSRecordHosts: record.Spec.DNSRecordHosts,
			IPSource:       ipSource,
			TTL:            ttl,
		},
	}

	// 4) Add the record as owner
	if err := ctrl.SetControllerReference(&record, &zoneRecord, r.scheme); err != nil {
		return v1alpha1.DNSZoneRecord{}, fmt.Errorf("Failed setting owner: %s", err)
	}

	return zoneRecord, nil
}

func (r *recordReconciler) deleteOwnedCertificatesNotMatchingName(
	record v1alpha1.DNSRecord, name string,
) error {
	ctx := context.Background()

	// Do not use the 'DeleteAll' function - it does not work with our custom indexers as the
	// request is sent directly to the Kubernetes API server. Also, we cannot use multiple field
	// selector parameters.z
	option := &client.ListOptions{
		Namespace: record.Namespace,
		FieldSelector: fields.AndSelectors(
			fields.OneTermEqualSelector(indexFieldOwner, record.Name),
		),
	}

	// 1) List certificates
	var list v1alpha2.CertificateList
	if err := r.client.List(ctx, &list, option); err != nil {
		return fmt.Errorf("Failed listing owned certificates: %s", err)
	}

	// 2) Delete all of them
	for _, certificate := range list.Items {
		if certificate.Name != name {
			if err := r.client.Delete(ctx, &certificate); err != nil {
				return fmt.Errorf("Failed deleting certificate: %s", err)
			}
		}
	}

	return nil
}

func (r *recordReconciler) certificate(record v1alpha1.DNSRecord) (v1alpha2.Certificate, error) {
	tls := *record.Spec.TLS

	// 1) Get DNS names
	dnsNames := make([]string, 0)
	for _, zone := range record.Spec.Zones {
		backend, ok := r.cache.Get(zone.Name)
		if !ok {
			return v1alpha2.Certificate{}, fmt.Errorf("Did not find zone '%s'", zone.Name)
		}

		// 1) Get key values
		for _, host := range record.Spec.Hosts {
			host = utils.FullHost(host, backend)
			dnsNames = append(dnsNames, host)
		}
		for _, cname := range record.Spec.Cnames {
			cname = utils.FullHost(cname, backend)
			dnsNames = append(dnsNames, cname)
		}
	}

	// 2) Create certificate
	certificate := v1alpha2.Certificate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      tls.CertificateName,
			Namespace: record.Namespace,
		},
		Spec: v1alpha2.CertificateSpec{
			DNSNames:   dnsNames,
			SecretName: tls.SecretName,
			IssuerRef: cmmetav1.ObjectReference{
				Name:  tls.Issuer.Name,
				Kind:  string(tls.Issuer.Kind),
				Group: "cert-manager.io",
			},
		},
	}

	// 3) Set owner
	if err := ctrl.SetControllerReference(&record, &certificate, r.scheme); err != nil {
		return v1alpha2.Certificate{}, fmt.Errorf("Failed setting owner: %s", err)
	}

	return certificate, nil
}

// UpdateIfNeeded sets the new status and returns whether any changes have occurred.
func certificateUpdateIfNeeded(old *v1alpha2.Certificate, new v1alpha2.Certificate) bool {
	oldIssuer := old.Spec.IssuerRef
	newIssuer := new.Spec.IssuerRef
	if old.Spec.SecretName == new.Spec.SecretName &&
		oldIssuer.Name == newIssuer.Name &&
		oldIssuer.Kind == newIssuer.Kind &&
		oldIssuer.Group == newIssuer.Group &&
		utils.Equal(old.Spec.DNSNames, new.Spec.DNSNames) {
		// No changes necessary
		return false
	}
	// Otherwise, we replace the entire specification
	old.Spec = new.Spec
	return true
}
