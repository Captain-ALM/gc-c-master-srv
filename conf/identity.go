package conf

type IdentityYaml struct {
	PublicKey  string `yaml:"publicKey"`
	PrivateKey string `yaml:"privateKey"`
	CanUpdate  bool   `yaml:"canUpdate"`
}
