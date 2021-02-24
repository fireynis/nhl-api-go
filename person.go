package nhlAPI

import (
	"encoding/json"
	"time"
)

type PersonsResponse struct {
	*BaseResponse
	Persons []Person
}

type Person struct {
	ID                 int       `json:"id"`
	FullName           string    `json:"fullName"`
	Link               string    `json:"link"`
	FirstName          string    `json:"firstName"`
	LastName           string    `json:"lastName"`
	PrimaryNumber      string    `json:"primaryNumber"`
	BirthDate          time.Time `json:"-"`
	CurrentAge         int       `json:"currentAge"`
	BirthCity          string    `json:"birthCity"`
	BirthStateProvince string    `json:"birthStateProvince"`
	BirthCountry       string    `json:"birthCountry"`
	Nationality        string    `json:"nationality"`
	Height             string    `json:"height"`
	Weight             int       `json:"weight"`
	Active             bool      `json:"active"`
	AlternateCaptain   bool      `json:"alternateCaptain"`
	Captain            bool      `json:"captain"`
	Rookie             bool      `json:"rookie"`
	ShootsCatches      string    `json:"shootsCatches"`
	RosterStatus       string    `json:"rosterStatus"`
	CurrentTeam        struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
		Link string `json:"link"`
	} `json:"currentTeam"`
	PrimaryPosition struct {
		Code         string `json:"code"`
		Name         string `json:"name"`
		Type         string `json:"type"`
		Abbreviation string `json:"abbreviation"`
	} `json:"primaryPosition"`
}

func (p *Person) UnmarshalJSON(bytes []byte) error {
	type PersonAlias Person
	alias := &struct {
		*PersonAlias
		DOB string `json:"birthDate"`
	}{
		PersonAlias: (*PersonAlias)(p),
	}

	if err := json.Unmarshal(bytes, alias); err != nil {
		return err
	}

	dob, err := time.Parse("2006-01-02", alias.DOB)
	if err != nil {
		return err
	}
	p.BirthDate = dob

	return nil
}

func (a *API) GetPerson(id int) (Person, error) {

}
