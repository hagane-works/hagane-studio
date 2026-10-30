package exportcheck

type BasemodTemplate struct {
	ID              int64    `json:"id"`
	ModKey          string   `json:"mod_key"`
	Version         string   `json:"version"`
	DisplayName     string   `json:"display_name"`
	Status          string   `json:"status"`
	WorkshopURL     *string  `json:"workshop_url"`
	DescriptorNames []string `json:"descriptor_names"`
	IndexDepKey     string   `json:"index_dep_key"`
	BucketPrefix    string   `json:"bucket_prefix"`
}
