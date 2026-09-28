package subnet

import (
	"errors"
	"net/http"
	"testing"

	"github.com/super-phenix/superphenix/internal/superphenix-api/internal/db/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

func TestCheckVpcReference(t *testing.T) {
	ownProject := uuid.New()
	otherProject := uuid.New()

	products := map[string]model.Product{
		"spx-own-vpc":    {ProjectId: ownProject, ProductTypeId: model.ProductTypeVPC.Name},
		"spx-other-vpc":  {ProjectId: otherProject, ProductTypeId: model.ProductTypeVPC.Name},
		"spx-own-subnet": {ProjectId: ownProject, ProductTypeId: model.ProductTypeSubnet.Name},
	}
	find := func(eid string) (model.Product, error) {
		if eid == "spx-db-error" {
			return model.Product{}, errors.New("connection refused")
		}
		p, ok := products[eid]
		if !ok {
			return model.Product{}, gorm.ErrRecordNotFound
		}
		return p, nil
	}

	tests := []struct {
		name     string
		vpcEId   string
		wantCode int
		wantErr  bool
	}{
		{name: "own VPC is accepted", vpcEId: "spx-own-vpc"},
		{name: "VPC of another project is rejected", vpcEId: "spx-other-vpc", wantCode: http.StatusNotFound, wantErr: true},
		{name: "own resource that is not a VPC is rejected", vpcEId: "spx-own-subnet", wantCode: http.StatusNotFound, wantErr: true},
		{name: "empty VPC is rejected", vpcEId: "", wantCode: http.StatusBadRequest, wantErr: true},
		{name: "VPC unknown to the database is left to the controller", vpcEId: "spx-unknown"},
		{name: "database error fails closed", vpcEId: "spx-db-error", wantCode: http.StatusInternalServerError, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, err := checkVpcReference(tt.vpcEId, ownProject, find)
			if (err != nil) != tt.wantErr {
				t.Fatalf("checkVpcReference() error = %v, wantErr %v", err, tt.wantErr)
			}
			if code != tt.wantCode {
				t.Errorf("checkVpcReference() code = %d, want %d", code, tt.wantCode)
			}
		})
	}
}
