package nhlAPI

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type PersonsResponse struct {
	*BaseResponse
	Persons []Person `json:"people"`
}

type Person struct {
	ID                 int             `json:"id"`
	FullName           string          `json:"fullName"`
	Link               string          `json:"link"`
	FirstName          string          `json:"firstName"`
	LastName           string          `json:"lastName"`
	PrimaryNumber      string          `json:"primaryNumber"`
	BirthDate          time.Time       `json:"-"`
	CurrentAge         int             `json:"currentAge"`
	BirthCity          string          `json:"birthCity"`
	BirthStateProvince string          `json:"birthStateProvince"`
	BirthCountry       string          `json:"birthCountry"`
	Nationality        string          `json:"nationality"`
	Height             string          `json:"height"`
	Weight             int             `json:"weight"`
	Active             bool            `json:"active"`
	AlternateCaptain   bool            `json:"alternateCaptain"`
	Captain            bool            `json:"captain"`
	Rookie             bool            `json:"rookie"`
	ShootsCatches      string          `json:"shootsCatches"`
	RosterStatus       string          `json:"rosterStatus"`
	CurrentTeam        CurrentTeam     `json:"currentTeam"`
	PrimaryPosition    PrimaryPosition `json:"primaryPosition"`
}

type CurrentTeam struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Link string `json:"link"`
}

type PrimaryPosition struct {
	Code         string `json:"code"`
	Name         string `json:"name"`
	Type         string `json:"type"`
	Abbreviation string `json:"abbreviation"`
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

func (api *API) GetPerson(id string) (*Person, error) {

	api.logger.Printf("URI %s", "/people/"+id)
	uri := "/people/" + id

	res, err := api.makeRequest("GET", uri, http.Header{})

	api.logger.Printf("Request has response")

	if err != nil {
		return nil, fmt.Errorf("unable to make request %w", err)
	}

	var personResponse PersonsResponse
	err = json.Unmarshal(res, &personResponse)
	if err != nil {
		return nil, fmt.Errorf("unable to unmarshall person %w", err)
	}

	if len(personResponse.Persons) > 1 {
		return nil, fmt.Errorf("unexpected result count: recieved more than one person from request")
	}

	if len(personResponse.Persons) < 1 {
		return nil, fmt.Errorf("no person found")
	}

	return &personResponse.Persons[0], nil
}
