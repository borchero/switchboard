package utils

import (
	"github.com/borchero/switchboard/api/v1alpha1"
	"github.com/mitchellh/hashstructure"
)

type void struct{}

// ResourceDiff computes the difference between the two given arrays of resources in terms of the
// contents of the specs. It returns elements that are not present in rhs as well as elements that
// are only present in rhs (i.e. insert and delete candidates).
func ResourceDiff(
	lhs, rhs []v1alpha1.DNSResource,
) ([]v1alpha1.DNSResource, []v1alpha1.DNSResource) {
	// 1) Setup lookup sets to prevent O(n^2) runtime
	lhsLookup := make(map[uint64]void)
	rhsLookup := make(map[uint64]void)

	for _, resource := range lhs {
		h, _ := hashstructure.Hash(resource.Spec, nil)
		lhsLookup[h] = void{}
	}

	for _, resource := range rhs {
		h, _ := hashstructure.Hash(resource.Spec, nil)
		rhsLookup[h] = void{}
	}

	// 2) Find elements that are contained in lhs but not in rhs
	missing := make([]v1alpha1.DNSResource, 0)
	for _, resource := range lhs {
		h, _ := hashstructure.Hash(resource.Spec, nil)
		if _, ok := rhsLookup[h]; !ok {
			missing = append(missing, resource)
		}
	}

	// 3) Find elements that are contained in rhs but not in lhs
	excess := make([]v1alpha1.DNSResource, 0)
	duplicateLookup := make(map[uint64]void) // we want to delete duplicates
	for _, resource := range rhs {
		h, _ := hashstructure.Hash(resource.Spec, nil)
		if _, ok := duplicateLookup[h]; ok {
			excess = append(excess, resource)
		} else if _, ok := lhsLookup[h]; !ok {
			excess = append(excess, resource)
		}
		duplicateLookup[h] = void{}
	}

	return missing, excess
}

// ZoneRecordDiff computes the difference between the two given arrays of zone records in terms of
// the contents of the specs. It returns elements that are not present in rhs as well as elements
// that are only present in rhs (i.e. insert and delete candidates).
func ZoneRecordDiff(
	lhs, rhs []v1alpha1.DNSZoneRecord,
) ([]v1alpha1.DNSZoneRecord, []v1alpha1.DNSZoneRecord) {
	// 1) Setup lookup sets to prevent O(n^2) runtime
	lhsLookup := make(map[uint64]void)
	rhsLookup := make(map[uint64]void)

	for _, record := range lhs {
		h, _ := hashstructure.Hash(record.Spec, nil)
		lhsLookup[h] = void{}
	}

	for _, record := range rhs {
		h, _ := hashstructure.Hash(record.Spec, nil)
		rhsLookup[h] = void{}
	}

	// 2) Find elements that are contained in lhs but not in rhs
	missing := make([]v1alpha1.DNSZoneRecord, 0)
	for _, record := range lhs {
		h, _ := hashstructure.Hash(record.Spec, nil)
		if _, ok := rhsLookup[h]; !ok {
			missing = append(missing, record)
		}
	}

	// 3) Find elements that are contained in rhs but not in lhs
	excess := make([]v1alpha1.DNSZoneRecord, 0)
	duplicateLookup := make(map[uint64]void) // we want to delete duplicates
	for _, record := range rhs {
		h, _ := hashstructure.Hash(record.Spec, nil)
		if _, ok := duplicateLookup[h]; ok {
			excess = append(excess, record)
		} else if _, ok := lhsLookup[h]; !ok {
			excess = append(excess, record)
		}
		duplicateLookup[h] = void{}
	}

	return missing, excess
}
