package operation_setting

import "github.com/QuantumNous/new-api/setting/config"

type ManualBankTransferConfig struct {
	Enabled       bool   `json:"enabled"`
	BankName      string `json:"bank_name"`
	BankCode      string `json:"bank_code"`
	BranchName    string `json:"branch_name"`
	AccountName   string `json:"account_name"`
	AccountNumber string `json:"account_number"`
	Instructions  string `json:"instructions"`
}

type manualBankTransferSetting struct {
	Config ManualBankTransferConfig `json:"config"`
}

var manualBankTransfer = manualBankTransferSetting{}

func init() {
	config.GlobalConfig.Register("manual_bank_transfer", &manualBankTransfer)
}

func GetManualBankTransferConfig() *ManualBankTransferConfig {
	return &manualBankTransfer.Config
}
