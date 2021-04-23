package parsers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pkg/errors"
)

// Error codes
var (
	ErrDBUnset  = errors.New("database not set")
	ErrNotFound = errors.New("not found")
)

type FIIStore struct {
	db *sql.DB
}

// NewFIIStore creates a new instace of FII.
func NewFIIStore(db *sql.DB) *FIIStore {
	fii := &FIIStore{
		db: db, // will accept null db when caching is no needed
	}
	return fii
}

// FIIDetails details (ID field: DetailFund.CNPJ)
type FIIDetails struct {
	DetailFund struct {
		Acronym               string      `json:"acronym"`
		TradingName           string      `json:"tradingName"`
		TradingCode           string      `json:"tradingCode"`
		TradingCodeOthers     string      `json:"tradingCodeOthers"`
		CNPJ                  string      `json:"cnpj"`
		Classification        string      `json:"classification"`
		WebSite               string      `json:"webSite"`
		FundAddress           string      `json:"fundAddress"`
		FundPhoneNumberDDD    string      `json:"fundPhoneNumberDDD"`
		FundPhoneNumber       string      `json:"fundPhoneNumber"`
		FundPhoneNumberFax    string      `json:"fundPhoneNumberFax"`
		PositionManager       string      `json:"positionManager"`
		ManagerName           string      `json:"managerName"`
		CompanyAddress        string      `json:"companyAddress"`
		CompanyPhoneNumberDDD string      `json:"companyPhoneNumberDDD"`
		CompanyPhoneNumber    string      `json:"companyPhoneNumber"`
		CompanyPhoneNumberFax string      `json:"companyPhoneNumberFax"`
		CompanyEmail          string      `json:"companyEmail"`
		CompanyName           string      `json:"companyName"`
		QuotaCount            string      `json:"quotaCount"`
		QuotaDateApproved     string      `json:"quotaDateApproved"`
		Codes                 []string    `json:"codes"`
		CodesOther            interface{} `json:"codesOther"`
		Segment               interface{} `json:"segment"`
	} `json:"detailFund"`
	ShareHolder struct {
		ShareHolderName           string `json:"shareHolderName"`
		ShareHolderAddress        string `json:"shareHolderAddress"`
		ShareHolderPhoneNumberDDD string `json:"shareHolderPhoneNumberDDD"`
		ShareHolderPhoneNumber    string `json:"shareHolderPhoneNumber"`
		ShareHolderFaxNumber      string `json:"shareHolderFaxNumber"`
		ShareHolderEmail          string `json:"shareHolderEmail"`
	} `json:"shareHolder"`
}

//
// StoreFIIDetails parses the stream data into FIIDetails and returns
// the *FIIDetails.
//
func (fii FIIStore) StoreFIIDetails(stream []byte) error {
	if fii.db == nil {
		return ErrDBUnset
	}

	if !hasTable(fii.db, "fii_details") {
		if err := createTable(fii.db, "fii_details"); err != nil {
			return err
		}
	}

	var fiiDetails FIIDetails
	if err := json.Unmarshal(stream, &fiiDetails); err != nil {
		return errors.Wrap(err, "json unmarshal")
	}

	trimFIIDetails(&fiiDetails)

	x := fiiDetails.DetailFund
	if x.CNPJ == "" {
		return fmt.Errorf("wrong CNPJ: %s", x.CNPJ)
	}

	insert := "INSERT OR IGNORE INTO fii_details (cnpj, acronym, trading_code) VALUES (?,?,?)"
	_, err := fii.db.Exec(insert, x.CNPJ, x.Acronym, x.TradingCode)

	return err
}

func (fii FIIStore) CNPJ(code string) (string, error) {
	if fii.db == nil {
		return "", ErrDBUnset
	}

	var query string
	if len(code) == 4 {
		query = `SELECT cnpj FROM fii_details WHERE acronym=?`
	} else if len(code) == 6 {
		query = `SELECT cnpj FROM fii_details WHERE trading_code=?`
	} else {
		return "", fmt.Errorf("invalid code '%s'", code)
	}

	var cnpj string
	row := fii.db.QueryRow(query, code)
	err := row.Scan(&cnpj)
	if err != nil && err != sql.ErrNoRows {
		return "", err
	}

	return cnpj, nil
}

func (fii FIIStore) SelectFIIDetails(code string) (*FIIDetails, error) {
	if fii.db == nil {
		return nil, ErrDBUnset
	}

	var query string
	if len(code) == 4 {
		query = `SELECT cnpj, acronym, trading_code FROM fii_details WHERE acronym=?`
	} else if len(code) == 6 {
		query = `SELECT cnpj, acronym, trading_code FROM fii_details WHERE trading_code=?`
	} else {
		return nil, fmt.Errorf("invalid code '%s'", code)
	}

	var cnpj, acronym, tradingCode string
	row := fii.db.QueryRow(query, code)
	err := row.Scan(&cnpj, &acronym, &tradingCode)
	if err != nil {
		return nil, err
	}

	var fiiDetail FIIDetails
	fiiDetail.DetailFund.CNPJ = cnpj
	fiiDetail.DetailFund.Acronym = acronym
	fiiDetail.DetailFund.TradingCode = tradingCode

	return &fiiDetail, nil
}

func trimFIIDetails(f *FIIDetails) {
	f.DetailFund.CNPJ = strings.TrimSpace(f.DetailFund.CNPJ)
	f.DetailFund.Acronym = strings.TrimSpace(f.DetailFund.Acronym)
	f.DetailFund.TradingCode = strings.TrimSpace(f.DetailFund.TradingCode)
}
