package main

import (
	"io/ioutil"
	"reflect"
	"testing"
)

func Test_getStoriesFromJSON(t *testing.T) {
	filename := "data/stories.json"
	want := []string{
		"https://www.familysearch.org/patron/v2/TH-303-46704-384-34/dist.txt?ctx=ArtCtxPublic",
		"https://www.familysearch.org/patron/v2/TH-300-46705-76-24/dist.txt?ctx=ArtCtxPublic",
		"https://www.familysearch.org/patron/v2/TH-303-46705-122-19/dist.txt?ctx=ArtCtxPublic",
		"https://www.familysearch.org/patron/v2/TH-300-46706-40-28/dist.txt?ctx=ArtCtxPublic",
		"https://www.familysearch.org/patron/v2/TH-303-46706-297-42/dist.txt?ctx=ArtCtxPublic",
	}
	fakeJSON, err := ioutil.ReadFile(filename)

	if err != nil {
		t.Errorf("Error reading fake data file %s", filename)
	}
	if got := getStoriesFromJSON(fakeJSON); !reflect.DeepEqual(got, want) {
		t.Errorf("getStoriesFromJSON() = %v, want %v", got, want)
	}
}

func Test_getPeopleFromJSON(t *testing.T) {
	filename := "data/people.json"
	want := []Person{
		Person{Name: "Ephraim Howard Kunz", ID: "KWFK-FVP"},
		Person{Name: "Howard William Kunz", ID: "LF7S-JD8"},
		Person{Name: "Tricia Joy Stockett", ID: "LF7S-J8H"},
		Person{Name: "Lyman Milton Kunz", ID: "KWZG-519"},
		Person{Name: "Opal Fern Hart", ID: "LF7S-JZ4"},
		Person{Name: "Don Carl Stockett", ID: "LF7S-J8R"},
		Person{Name: "Laurel Claire Wells", ID: "LF7S-JCG"},
		Person{Name: "Heber Christian Kunz", ID: "KWCF-6DZ"},
	}
	fakeJSON, err := ioutil.ReadFile(filename)

	if err != nil {
		t.Errorf("Error reading fake data file %s", filename)
	}
	if got := getPeopleFromJSON(fakeJSON); !reflect.DeepEqual(got, want) {
		t.Errorf("getStoriesFromJSON() = %v, want %v", got, want)
	}
}
