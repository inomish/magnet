package generate

//go:generate mkdir -p generated/protocol
//go:generate go tool wlclgen -package protocol -out generated/protocol/gen.go ./protocol/*.xml
