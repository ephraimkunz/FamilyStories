package main

import (
	"math/rand"
)

func shufflePeople(in []Person, r *rand.Rand) []Person {
	dest := make([]Person, len(in))
	perm := r.Perm(len(in))
	for i, v := range perm {
		dest[v] = in[i]
	}
	return dest
}

func shuffleStrings(in []string, r *rand.Rand) []string {
	dest := make([]string, len(in))
	perm := r.Perm(len(in))
	for i, v := range perm {
		dest[v] = in[i]
	}
	return dest
}
