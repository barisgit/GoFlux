package generator

// ClientTemplateData contains data for client template generation
type ClientTemplateData struct {
	UsedTypes         []string
	TypesImport       string
	APIObject         string
	ReactQueryEnabled bool
	QueryKeysEnabled  bool
	QueryKeys         string
}

// MethodTemplateData contains data for individual method templates
type MethodTemplateData struct {
	Description                    string
	Method                         string
	MethodLower                    string
	ParameterSignature             string
	ParameterSignatureJS           string // JavaScript parameter signature with destructuring
	QueryParameterSignature        string
	QueryOptionsParameterSignature string
	ResponseType                   string
	RequestPath                    string
	RequestPathForMutation         string // For React Query mutations with different variable substitution
	HasIDParam                     bool
	HasBodyData                    bool
	DataParameter                  string
	QueryKey                       string
	MutationVariableType           string
	ReactQueryEnabled              bool
}
