dep:
	go get github.com/mitchellh/gox

gox:
	gox \
		-os="darwin linux windows" \
		-arch="386 amd64" \
		-output "pkg/{{.OS}}_{{.Arch}}/{{.Dir}}"
