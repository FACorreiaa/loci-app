package assets

import "embed"

//go:embed css/* js/* static/*
var Assets embed.FS
