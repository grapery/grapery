package compliance

var (
	GlobalComplianceTool *ComplianceTool
)

func GetComplianceTool() *ComplianceTool {
	return GlobalComplianceTool
}

type ComplianceTool struct {
	Address string
	Secret  string
}

func init() {
	GlobalComplianceTool = &ComplianceTool{}
}

func Init(address string, secret string) *ComplianceTool {
	return &ComplianceTool{
		Address: address,
		Secret:  secret,
	}
}

func (*ComplianceTool) TextCompliance(content string) error {
	return nil
}

func (*ComplianceTool) ImageCompliance(image string) error {
	return nil
}
