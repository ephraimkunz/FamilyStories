package main

import (
	"math/rand"
	"reflect"
	"testing"
)

func Test_shufflePeople(t *testing.T) {
	r := rand.New(rand.NewSource(1))
	people := []Person{
		Person{Name: "Jim", ID: "UIK-EWQ"},
		Person{Name: "Ephraim", ID: "KFDO-s"},
		Person{Name: "Howard", ID: "IDGOVE-"},
		Person{Name: "Megan", ID: "DVSK-534"},
		Person{Name: "FEBE", ID: "435j-f"},
	}

	type args struct {
		in []Person
		r  *rand.Rand
	}
	tests := []struct {
		name string
		args args
		want []Person
	}{
		{"Test shuffle 1", args{[]Person{people[0], people[1], people[2], people[3], people[4]}, r}, []Person{people[0], people[4], people[2], people[3], people[1]}},
		{"Test shuffle 2", args{[]Person{people[0], people[4], people[2], people[3], people[1]}, r}, []Person{people[3], people[4], people[2], people[1], people[0]}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shufflePeople(tt.args.in, tt.args.r); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("shufflePeople() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_shuffleStrings(t *testing.T) {
	r := rand.New(rand.NewSource(1))

	type args struct {
		in []string
		r  *rand.Rand
	}
	tests := []struct {
		name string
		args args
		want []string
	}{
		{"Test shuffle 1", args{[]string{"a", "b", "c", "d", "e"}, r}, []string{"a", "e", "c", "d", "b"}},
		{"Test shuffle 2", args{[]string{"a", "e", "c", "d", "b"}, r}, []string{"d", "e", "c", "b", "a"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shuffleStrings(tt.args.in, tt.args.r); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("shuffleStrings() = %v, want %v", got, tt.want)
			}
		})
	}
}
