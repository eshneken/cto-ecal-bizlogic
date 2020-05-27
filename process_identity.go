//  ProcessIdentity
//	CTO Business Logic Helpers
//	Ed Shnekendorf, 2020, https://github.com/eshneken/cto-bizlogic-helper
//
// refer to https://golang.org/src/database/sql/sql_test.go for golang SQL samples

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

// Employee represents an individual returned from the corporate feed
type Employee struct {
	ID                   string `json:"id"`
	EmployeeEmailAddress string `json:"employee_email_address"`
	Role                 string `json:"role"`
	Status               string `json:"status"`
	RecordType           string `json:"record_type"`
	Title                string `json:"title"`
	Mgr                  string `json:"mgr"`
	Lob                  string `json:"lob"`
	CostCenter           string `json:"cost_center"`
	Region               string `json:"region"`
	Country              string `json:"country"`
	StartDate            string `json:"start_date"`
	EndDate              string `json:"end_date"`
	CreatedOn            string `json:"created_on"`
	CreatedBy            string `json:"created_by"`
	UpdatedOn            string `json:"updated_on"`
	UpdatedBy            string `json:"updated_by"`
	EmployeeFullName     string `json:"employee_full_name"`
	LdapStatus           string `json:"ldap_status"`
	Evp                  string `json:"evp"`
	EvpDirect            string `json:"evp_direct"`
	NeverProcessLdap     string `json:"never_process_ldap"`
	DoNotUpdateFromLdap  string `json:"do_not_update_from_ldap"`
	LockRegion           string `json:"lock_region"`
	LeftCompanyOn        string `json:"left_company_on"`
	Inactive             string `json:"inactive"`
	MgrLevel             string `json:"mgr_level"`
	State                string `json:"state"`
	City                 string `json:"city"`
	MgrChain             string `json:"mgr_chain"`
	TopMgrDirMinus1      string `json:"top_mgr_dir_minus_1"`
	TopMgrDirMinus2      string `json:"top_mgr_dir_minus_2"`
	TopMgrDirMinus3      string `json:"top_mgr_dir_minus_3"`
	TopMgrDirMinus4      string `json:"top_mgr_dir_minus_4"`
	NumDirects           string `json:"num_directs"`
	NumUsers             string `json:"num_users"`
	OldUID               string `json:"olduid"`
	ChainLevel           string `json:"chain_level"`
	OracleUID            string `json:"oracle_uid"`
	LobDetail            string `json:"lob_detail"`
	HierLevel            string `json:"hier_level"`
	TopMgrSeq            string `json:"top_mgr_seq"`
}

func processIdentity(filename string) {
	file, err := os.Open(filename)
	if err != nil {
		fmt.Printf("[%s] processIdentity: Error opening file [%s]: %s\n",
			time.Now().Format(time.RFC3339), filename, err.Error())
		return
	}
	defer file.Close()

	// seek 10 bytes (chars) to advance past {"items":
	_, err = file.Seek(10, io.SeekStart)
	if err != nil {
		fmt.Printf("[%s] processIdentity: Error advancing file stream to position 10: %s\n",
			time.Now().Format(time.RFC3339), err.Error())
		return
	}

	// create a JSON stream decoder
	decoder := json.NewDecoder(file)
	fmt.Printf("[%s] processIdentity: START Processing identities\n", time.Now().Format(time.RFC3339))

	// start a DB transaction
	tx, err := DBPool.Begin()
	defer tx.Rollback()
	if err != nil {
		fmt.Printf("[%s] processIdentity: Error starting DB transaction: %s\n",
			time.Now().Format(time.RFC3339), err.Error())
		return
	}

	// delete all data from LookupOpportunity table
	_, err = tx.Exec("DELETE FROM CTO_COMMON.ORACLE_EMPLOYEES")
	if err != nil {
		fmt.Printf("[%s] processIdentity: Error deleting from CTO_COMMON.ORACLE_EMPLOYEES: %s\n",
			time.Now().Format(time.RFC3339), err.Error())
		return
	}

	// prepare insert statement
	query := `INSERT INTO CTO_COMMON.ORACLE_EMPLOYEES (
		ID, 
		EMPLOYEE_EMAIL_ADDRESS, 
		ROLE, 
		STATUS, 
		RECORD_TYPE, 
		TITLE, 
		MGR, 
		LOB, 
		COST_CENTER, 
		REGION, 
		COUNTRY, 
		START_DATE, 
		END_DATE, 
		CREATED_ON, 
		CREATED_BY, 
		UPDATED_ON, 
		UPDATED_BY, 
		EMPLOYEE_FULL_NAME, 
		LDAP_STATUS, 
		EVP, 
		EVP_DIRECT, 
		NEVER_PROCESS_LDAP, 
		DO_NOT_UPDATE_FROM_LDAP, 
		LOCK_REGION, 
		LEFT_COMPANY_ON, 
		INACTIVE, 
		MGR_LEVEL, 
		STATE, 
		CITY, 
		MGR_CHAIN, 
		TOP_MGR_DIR_MINUS_1, 
		TOP_MGR_DIR_MINUS_2, 
		TOP_MGR_DIR_MINUS_3, 
		TOP_MGR_DIR_MINUS_4, 
		NUM_DIRECTS, 
		NUM_USERS, 
		OLDUID, 
		CHAIN_LEVEL, 
		ORACLE_UID, 
		LOB_DETAIL, 
		HIER_LEVEL, 
		TOP_MGR_SEQ 
	) VALUES (
		TO_NUMBER(:1), :2, :3, :4, :5,:6, :7, :8, :9, :10, :11, TO_DATE(:12, 'YYYY-MM-DD'), 
		TO_DATE(:13, 'YYYY-MM-DD'), TO_DATE(:14, 'YYYY-MM-DD'), :15, 
		TO_DATE(:16, 'YYYY-MM-DD'), :17, :18, :19, :20, :21, :22, :23, :24, 
		TO_DATE(:25, 'YYYY-MM-DD'), TO_DATE(:26, 'YYYY-MM-DD'), :27, :28, 
		:29, :30, :31, :32, :33, :34, TO_NUMBER(:35), TO_NUMBER(:36), :37, TO_NUMBER(:38), :39, :40, TO_NUMBER(:41), TO_NUMBER(:42)
	)
	`
	insertStmt, err := tx.Prepare(query)
	defer insertStmt.Close()
	if err != nil {
		fmt.Printf("[%s] processIdentity: Error preparing insert statement: %s\n",
			time.Now().Format(time.RFC3339), err.Error())
		return
	}

	// consume the opening array brace
	_, err = decoder.Token()
	if err != nil {
		fmt.Printf("[%s] processIdentity: Error decoding opening array token: %s\n",
			time.Now().Format(time.RFC3339), err.Error())
		return
	}

	// iterate each employee
	insertedEmps := 0
	counter := 1
	for decoder.More() {
		var person Employee
		err := decoder.Decode(&person)
		if err != nil {
			fmt.Printf("[%s] processIdentity: Error decoding person %d: %s\n",
				time.Now().Format(time.RFC3339), counter, err.Error())
			return
		}
		counter++

		// truncate timestamps
		person.StartDate = strings.TrimSuffix(strings.Split(person.StartDate, "T")[0], "T")
		person.EndDate = strings.TrimSuffix(strings.Split(person.EndDate, "T")[0], "T")
		person.CreatedOn = strings.TrimSuffix(strings.Split(person.CreatedOn, "T")[0], "T")
		person.UpdatedOn = strings.TrimSuffix(strings.Split(person.UpdatedOn, "T")[0], "T")
		person.LeftCompanyOn = strings.TrimSuffix(strings.Split(person.LeftCompanyOn, "T")[0], "T")
		person.Inactive = strings.TrimSuffix(strings.Split(person.Inactive, "T")[0], "T")

		if person.Lob != "X-LEFT ORACLE" {
			_, err = insertStmt.Exec(person.ID, person.EmployeeEmailAddress, person.Role, person.Status, person.RecordType,
				person.Title, person.Mgr, person.Lob, person.CostCenter, person.Region, person.Country, person.StartDate,
				person.EndDate, person.CreatedOn, person.CreatedBy, person.UpdatedOn, person.UpdatedBy, person.EmployeeFullName,
				person.LdapStatus, person.Evp, person.EvpDirect, person.NeverProcessLdap, person.DoNotUpdateFromLdap,
				person.LockRegion, person.LeftCompanyOn, person.Inactive, person.MgrLevel, person.State, person.City,
				person.MgrChain, person.TopMgrDirMinus1, person.TopMgrDirMinus2, person.TopMgrDirMinus3, person.TopMgrDirMinus4,
				person.NumDirects, person.NumUsers, person.OldUID, person.ChainLevel, person.OracleUID, person.LobDetail,
				person.HierLevel, person.TopMgrSeq)
			if err != nil {
				fmt.Printf("[%s] processIdentity: Error inserting person [%s]: %s\n",
					time.Now().Format(time.RFC3339), person.EmployeeFullName, err.Error())
				return
			}
			insertedEmps++
		}
	}

	// consume the closing array brace
	_, err = decoder.Token()
	if err != nil {
		fmt.Printf("[%s] processIdentity: Error decoding closing array token: %s\n",
			time.Now().Format(time.RFC3339), err.Error())
		return
	}

	// complete the transaction
	err = tx.Commit()
	if err != nil {
		fmt.Printf("[%s] processIdentity: Error committing transaction: %s\n",
			time.Now().Format(time.RFC3339), err.Error())
		return
	}

	fmt.Printf("[%s] processIdentity: DONE processing %d employees and loading %d current employees\n",
		time.Now().Format(time.RFC3339), counter, insertedEmps)
}