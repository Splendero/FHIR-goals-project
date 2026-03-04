package patient

import (
	"testing"

	"github.com/spencerosborn/fhir-goals-engine/internal/fhir"
	"github.com/stretchr/testify/assert"
)

func TestValidatePatient(t *testing.T) {
	tests := []struct {
		name    string
		patient Patient
		wantErr error
	}{
		{
			name:    "no name slice at all",
			patient: Patient{Name: nil},
			wantErr: ErrNameRequired,
		},
		{
			name:    "empty name slice",
			patient: Patient{Name: []fhir.HumanName{}},
			wantErr: ErrNameRequired,
		},
		{
			name: "name entry with empty family and no given",
			patient: Patient{
				Name: []fhir.HumanName{{Family: "", Given: nil}},
			},
			wantErr: ErrNameRequired,
		},
		{
			name: "name entry with empty family and empty given slice",
			patient: Patient{
				Name: []fhir.HumanName{{Family: "", Given: []string{}}},
			},
			wantErr: ErrNameRequired,
		},
		{
			name: "valid - family name only",
			patient: Patient{
				Name: []fhir.HumanName{{Family: "Osborn"}},
			},
			wantErr: nil,
		},
		{
			name: "valid - given name only",
			patient: Patient{
				Name: []fhir.HumanName{{Given: []string{"Spencer"}}},
			},
			wantErr: nil,
		},
		{
			name: "valid - both family and given",
			patient: Patient{
				Name: []fhir.HumanName{{Family: "Osborn", Given: []string{"Spencer"}}},
			},
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePatient(&tt.patient)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
