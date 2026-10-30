package exportengine

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

const (
	BuildIdentitySchemaVersion = "hagane.build-identity.v1"
	BuildIdentityFilename      = "_HAGANE_BUILD_ID.json"
)

func NewBuildIdentity() (string, error) {
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", fmt.Errorf("generate build identity: %w", err)
	}
	raw[6] = (raw[6] & 0x0f) | 0x40
	raw[8] = (raw[8] & 0x3f) | 0x80
	encoded := hex.EncodeToString(raw[:])
	return "hagane-build-" + strings.Join([]string{
		encoded[0:8], encoded[8:12], encoded[12:16], encoded[16:20], encoded[20:32],
	}, "-"), nil
}

func ApplyBuildIdentity(artifact *NativeArtifact, buildID string) error {
	if artifact == nil || strings.TrimSpace(buildID) == "" {
		return nil
	}
	payload := struct {
		BuildID       string `json:"build_id"`
		SchemaVersion string `json:"schema_version"`
	}{BuildID: buildID, SchemaVersion: BuildIdentitySchemaVersion}
	body, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	body = append(body, '\n')
	entries, err := overlayEntries(artifact.Entries, []Entry{{
		Path: artifactRoot(*artifact) + "/" + BuildIdentityFilename, Bytes: body,
		Origin: "build_identity", SourceEntity: "project",
	}})
	if err != nil {
		return err
	}
	artifact.Entries = entries
	artifact.Coverage = append(artifact.Coverage, "build_identity")
	return nil
}
