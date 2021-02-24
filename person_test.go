package nhlAPI

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

var personBytes = []byte("{\n  \"id\": 8447400,\n  \"fullName\": \"Wayne Gretzky\",\n  \"link\": \"/api/v1/people/8447400\",\n  \"firstName\": \"Wayne\",\n  \"lastName\": \"Gretzky\",\n  \"primaryNumber\": \"99\",\n  \"birthDate\": \"1961-01-26\",\n  \"birthCity\": \"Brantford\",\n  \"birthStateProvince\": \"ON\",\n  \"birthCountry\": \"CAN\",\n  \"nationality\": \"CAN\",\n  \"height\": \"6' 0\\\"\",\n  \"weight\": 185,\n  \"active\": false,\n  \"rookie\": false,\n  \"shootsCatches\": \"L\",\n  \"rosterStatus\": \"N\",\n  \"primaryPosition\": {\n    \"code\": \"C\",\n    \"name\": \"Center\",\n    \"type\": \"Forward\",\n    \"abbreviation\": \"C\"\n  }\n}")
var expectedDate = time.Date(1961, 01, 26, 0, 0, 0, 0, time.UTC)

func TestAPI_GetPerson(t *testing.T) {
	api, _ := New()
	type args struct {
		id string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{name: "Get Gretzky", args: struct{ id string }{id: "8447400"}, want: "Wayne Gretzky", wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			got, err := api.GetPerson(tt.args.id)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetPerson() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got == nil {
				t.Errorf("GetPerson() is nil!")
				return
			}
			if !reflect.DeepEqual(got.FullName, tt.want) {
				t.Errorf("GetPerson() got = %s, want %s", got.FullName, tt.want)
			}
		})
	}
}

func TestPerson_UnmarshalJSON(t *testing.T) {
	type args struct {
		bytes []byte
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{name: "Successful date parse", args: args{bytes: personBytes}, wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var p Person
			if err := json.Unmarshal(tt.args.bytes, &p); (err != nil) != tt.wantErr {
				t.Errorf("UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !reflect.DeepEqual(expectedDate, p.BirthDate) {
				t.Errorf("UnmarshalJSON() got = %v, wanted %v", p.BirthDate, expectedDate)
			}
		})
	}
}
