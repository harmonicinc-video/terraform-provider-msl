package models

import (
	"encoding/json"
	"testing"
	"time"
)

// ---- StreamFormat / StreamStatus / IngestAuthMode constants ----

func TestStreamFormat_Values(t *testing.T) {
	cases := []struct {
		fmt  StreamFormat
		want string
	}{
		{StreamFormatHLS, "HLS"},
		{StreamFormatCMAF, "CMAF"},
		{StreamFormatDASH, "DASH"},
	}
	for _, tc := range cases {
		if string(tc.fmt) != tc.want {
			t.Errorf("StreamFormat value = %q, want %q", tc.fmt, tc.want)
		}
	}
}

func TestStreamStatus_Values(t *testing.T) {
	cases := []struct {
		status StreamStatus
		want   string
	}{
		{StreamStatusCreating, "CREATING"},
		{StreamStatusReady, "READY"},
		{StreamStatusDeleted, "DELETED"},
	}
	for _, tc := range cases {
		if string(tc.status) != tc.want {
			t.Errorf("StreamStatus value = %q, want %q", tc.status, tc.want)
		}
	}
}

func TestIngestAuthMode_Values(t *testing.T) {
	cases := []struct {
		mode IngestAuthMode
		want string
	}{
		{IngestAuthModeNone, "NONE"},
		{IngestAuthModeDigest, "DIGEST"},
		{IngestAuthModeHeader, "HEADER"},
	}
	for _, tc := range cases {
		if string(tc.mode) != tc.want {
			t.Errorf("IngestAuthMode value = %q, want %q", tc.mode, tc.want)
		}
	}
}

// ---- Stream JSON round-trip ----

func TestStream_JSONRoundTrip(t *testing.T) {
	now := time.Date(2024, 6, 1, 9, 0, 0, 0, time.UTC)
	backup := "backup.example.com"
	backupHost := "backup-host.example.com"
	backupOrigin := "backup-origin.example.com"
	ingestAuth := true
	authMode := IngestAuthModeDigest
	playlist := int64(30)

	orig := Stream{
		ID:                       "stream-1",
		Description:              "test stream",
		Format:                   StreamFormatHLS,
		Status:                   StreamStatusReady,
		OriginID:                 "origin-1",
		IngestLocation:           "US_ORD",
		BackupIngestLocation:     &backup,
		HostName:                 "host.example.com",
		BackupHostName:           &backupHost,
		OriginHostName:           "origin.example.com",
		BackupOriginHostName:     &backupOrigin,
		PrimaryPublishingURL:     "rtmp://primary.example.com/live",
		BackupPublishingURL:      "rtmp://backup.example.com/live",
		PlaylistDurationInMin:    &playlist,
		AllowedIPs:               []string{"10.0.0.1", "192.168.1.0/24"},
		IngestAuthentication:     &ingestAuth,
		IngestAuthenticationMode: &authMode,
		ContractID:               "ctr-1",
		CPTag:                    "tag1",
		GroupID:                  "grp-1",
		CreatedBy:                "admin@example.com",
		CreatedAt:                now,
		UpdatedAt:                now,
	}

	data, err := json.Marshal(orig)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}

	var got Stream
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("json.Unmarshal() error: %v", err)
	}

	if got.ID != orig.ID {
		t.Errorf("ID = %q, want %q", got.ID, orig.ID)
	}
	if got.Format != orig.Format {
		t.Errorf("Format = %q, want %q", got.Format, orig.Format)
	}
	if got.Status != orig.Status {
		t.Errorf("Status = %q, want %q", got.Status, orig.Status)
	}
	if got.BackupIngestLocation == nil || *got.BackupIngestLocation != backup {
		t.Errorf("BackupIngestLocation = %v, want %q", got.BackupIngestLocation, backup)
	}
	if got.PlaylistDurationInMin == nil || *got.PlaylistDurationInMin != playlist {
		t.Errorf("PlaylistDurationInMin = %v, want %d", got.PlaylistDurationInMin, playlist)
	}
	if len(got.AllowedIPs) != 2 {
		t.Fatalf("len(AllowedIPs) = %d, want 2", len(got.AllowedIPs))
	}
	if got.IngestAuthenticationMode == nil || *got.IngestAuthenticationMode != authMode {
		t.Errorf("IngestAuthenticationMode = %v, want %q", got.IngestAuthenticationMode, authMode)
	}
}

func TestStream_JSONFieldNames(t *testing.T) {
	s := Stream{
		ID:         "s1",
		Format:     StreamFormatCMAF,
		OriginID:   "o1",
		ContractID: "c1",
		CPTag:      "t1",
		GroupID:    "g1",
	}
	data, err := json.Marshal(s)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}

	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("json.Unmarshal() error: %v", err)
	}

	for _, want := range []string{"stream_id", "format", "origin_id", "contract_id", "cptag", "group_id"} {
		if _, ok := m[want]; !ok {
			t.Errorf("JSON key %q missing from marshalled Stream", want)
		}
	}
}

// ---- Archiving ----

func TestArchiving_JSONRoundTrip(t *testing.T) {
	days := int64(7)
	a := Archiving{
		NoArchive:      false,
		AutomaticPurge: &AutomaticPurge{RetentionDays: &days},
	}

	data, err := json.Marshal(a)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}

	var got Archiving
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("json.Unmarshal() error: %v", err)
	}

	if got.NoArchive != a.NoArchive {
		t.Errorf("NoArchive = %v, want %v", got.NoArchive, a.NoArchive)
	}
	if got.AutomaticPurge == nil || got.AutomaticPurge.RetentionDays == nil {
		t.Fatal("AutomaticPurge.RetentionDays is nil")
	}
	if *got.AutomaticPurge.RetentionDays != days {
		t.Errorf("RetentionDays = %d, want %d", *got.AutomaticPurge.RetentionDays, days)
	}
}

func TestArchiving_NoArchiveOmitsPurge(t *testing.T) {
	a := Archiving{NoArchive: true}
	data, err := json.Marshal(a)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}

	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("json.Unmarshal() error: %v", err)
	}

	if _, ok := m["automatic_purge"]; ok {
		t.Error("automatic_purge should be omitted when nil")
	}
}

// ---- Playback / AkamaiG2oAuth ----

func TestPlayback_JSONRoundTrip(t *testing.T) {
	enabled := true
	version := int64(2)
	secret := "mysecret"
	delta := int64(30)
	p := Playback{
		AkamaiG2oAuth: &AkamaiG2oAuth{
			Enabled:    &enabled,
			G2oVersion: &version,
			SecretKey:  &secret,
			TimeDelta:  &delta,
		},
	}

	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}

	var got Playback
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("json.Unmarshal() error: %v", err)
	}

	if got.AkamaiG2oAuth == nil {
		t.Fatal("AkamaiG2oAuth is nil after unmarshal")
	}
	if got.AkamaiG2oAuth.SecretKey == nil || *got.AkamaiG2oAuth.SecretKey != secret {
		t.Errorf("SecretKey = %v, want %q", got.AkamaiG2oAuth.SecretKey, secret)
	}
	if got.AkamaiG2oAuth.G2oVersion == nil || *got.AkamaiG2oAuth.G2oVersion != version {
		t.Errorf("G2oVersion = %v, want %d", got.AkamaiG2oAuth.G2oVersion, version)
	}
}

func TestPlayback_NilG2oAuth(t *testing.T) {
	p := Playback{}
	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}

	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("json.Unmarshal() error: %v", err)
	}

	if _, ok := m["akamai_g2o_auth"]; ok {
		t.Error("akamai_g2o_auth should be omitted when nil")
	}
}

// ---- StreamIngestHeader ----

func TestStreamIngestHeader_JSONRoundTrip(t *testing.T) {
	header := "X-Auth-Token"
	h := StreamIngestHeader{
		Header: &header,
		Values: []string{"token1", "token2"},
	}

	data, err := json.Marshal(h)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}

	var got StreamIngestHeader
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("json.Unmarshal() error: %v", err)
	}

	if got.Header == nil || *got.Header != header {
		t.Errorf("Header = %v, want %q", got.Header, header)
	}
	if len(got.Values) != 2 {
		t.Fatalf("len(Values) = %d, want 2", len(got.Values))
	}
}

// ---- HlsToLlHlsConfig ----

func TestHlsToLlHlsConfig_JSONRoundTrip(t *testing.T) {
	cfg := HlsToLlHlsConfig{
		Enabled:         true,
		SegmentTemplate: "seg_$Number$.m4s",
	}

	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}

	var got HlsToLlHlsConfig
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("json.Unmarshal() error: %v", err)
	}

	if got.Enabled != cfg.Enabled {
		t.Errorf("Enabled = %v, want %v", got.Enabled, cfg.Enabled)
	}
	if got.SegmentTemplate != cfg.SegmentTemplate {
		t.Errorf("SegmentTemplate = %q, want %q", got.SegmentTemplate, cfg.SegmentTemplate)
	}
}

// ---- StreamCreateRequest ----

func TestStreamCreateRequest_JSONRoundTrip(t *testing.T) {
	backup := "backup-ingest"
	authMode := IngestAuthModeHeader
	ingestAuth := true
	playlist := int64(15)
	header := "X-Custom"
	req := StreamCreateRequest{
		Description:              "live stream",
		Format:                   StreamFormatDASH,
		OriginID:                 "origin-1",
		IngestLocation:           "GB_LON",
		BackupIngestLocation:     &backup,
		ContractID:               "ctr-2",
		CPTag:                    "tag2",
		GroupID:                  "grp-2",
		Archiving:                Archiving{NoArchive: true},
		PlaylistDurationInMin:    &playlist,
		AllowedIPs:               []string{"203.0.113.0/24"},
		IngestAuthentication:     &ingestAuth,
		IngestAuthenticationMode: &authMode,
		IngestHeader:             &StreamIngestHeader{Header: &header, Values: []string{"v1"}},
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}

	var got StreamCreateRequest
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("json.Unmarshal() error: %v", err)
	}

	if got.Format != req.Format {
		t.Errorf("Format = %q, want %q", got.Format, req.Format)
	}
	if got.BackupIngestLocation == nil || *got.BackupIngestLocation != backup {
		t.Errorf("BackupIngestLocation = %v, want %q", got.BackupIngestLocation, backup)
	}
	if got.Archiving.NoArchive != true {
		t.Errorf("Archiving.NoArchive = false, want true")
	}
	if got.IngestAuthenticationMode == nil || *got.IngestAuthenticationMode != authMode {
		t.Errorf("IngestAuthenticationMode = %v, want %q", got.IngestAuthenticationMode, authMode)
	}
	if got.IngestHeader == nil || got.IngestHeader.Header == nil || *got.IngestHeader.Header != header {
		t.Errorf("IngestHeader.Header = %v, want %q", got.IngestHeader, header)
	}
}

func TestStreamCreateRequest_OmitsEmptyOptionalFields(t *testing.T) {
	req := StreamCreateRequest{
		Format:         StreamFormatHLS,
		OriginID:       "o1",
		IngestLocation: "US_ORD",
		ContractID:     "c1",
		CPTag:          "t1",
		GroupID:        "g1",
		Archiving:      Archiving{NoArchive: true},
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}

	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("json.Unmarshal() error: %v", err)
	}

	for _, absent := range []string{"backup_ingest_location", "playback", "playlist_duration_in_min",
		"ingest_authentication", "ingest_authentication_mode", "ingest_header", "hls_to_llhls"} {
		if _, ok := m[absent]; ok {
			t.Errorf("JSON key %q should be omitted when nil/unset", absent)
		}
	}
}

// ---- StreamUpdateRequest ----

func TestStreamUpdateRequest_AllowedIPsAlwaysPresent(t *testing.T) {
	// allowed_ips uses json:"allowed_ips" (no omitempty) so it must appear even when empty.
	req := StreamUpdateRequest{}
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}

	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("json.Unmarshal() error: %v", err)
	}

	if _, ok := m["allowed_ips"]; !ok {
		t.Error("allowed_ips must always be present in StreamUpdateRequest (no omitempty)")
	}
}

func TestStreamUpdateRequest_OmitsNilOptionalFields(t *testing.T) {
	req := StreamUpdateRequest{}
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}

	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("json.Unmarshal() error: %v", err)
	}

	for _, absent := range []string{"description", "group_id", "archiving", "playback",
		"playlist_duration_in_min", "ingest_authentication", "ingest_authentication_mode",
		"ingest_header", "hls_to_llhls"} {
		if _, ok := m[absent]; ok {
			t.Errorf("JSON key %q should be omitted when nil", absent)
		}
	}
}

func TestStreamUpdateRequest_JSONRoundTrip(t *testing.T) {
	desc := "updated stream"
	groupID := "grp-new"
	days := int64(14)
	req := StreamUpdateRequest{
		Description: &desc,
		GroupID:     &groupID,
		Archiving:   &Archiving{NoArchive: false, AutomaticPurge: &AutomaticPurge{RetentionDays: &days}},
		AllowedIPs:  []string{"10.1.2.3"},
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("json.Marshal() error: %v", err)
	}

	var got StreamUpdateRequest
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("json.Unmarshal() error: %v", err)
	}

	if got.Description == nil || *got.Description != desc {
		t.Errorf("Description = %v, want %q", got.Description, desc)
	}
	if got.GroupID == nil || *got.GroupID != groupID {
		t.Errorf("GroupID = %v, want %q", got.GroupID, groupID)
	}
	if got.Archiving == nil || got.Archiving.AutomaticPurge == nil {
		t.Fatal("Archiving.AutomaticPurge is nil")
	}
	if *got.Archiving.AutomaticPurge.RetentionDays != days {
		t.Errorf("RetentionDays = %d, want %d", *got.Archiving.AutomaticPurge.RetentionDays, days)
	}
	if len(got.AllowedIPs) != 1 || got.AllowedIPs[0] != "10.1.2.3" {
		t.Errorf("AllowedIPs = %v, want [10.1.2.3]", got.AllowedIPs)
	}
}

// ---- ListStreamsResponse ----

func TestListStreamsResponse_Unmarshal(t *testing.T) {
	raw := `{"streams":[{"stream_id":"s1","format":"HLS","origin_id":"o1","ingest_location":"US_ORD","contract_id":"c1","cptag":"t1","group_id":"g1"},{"stream_id":"s2","format":"CMAF","origin_id":"o2","ingest_location":"GB_LON","contract_id":"c2","cptag":"t2","group_id":"g2"}]}`
	var resp ListStreamsResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		t.Fatalf("json.Unmarshal() error: %v", err)
	}
	if len(resp.Streams) != 2 {
		t.Fatalf("len(Streams) = %d, want 2", len(resp.Streams))
	}
	if resp.Streams[0].ID != "s1" {
		t.Errorf("Streams[0].ID = %q, want %q", resp.Streams[0].ID, "s1")
	}
	if resp.Streams[1].Format != StreamFormatCMAF {
		t.Errorf("Streams[1].Format = %q, want %q", resp.Streams[1].Format, StreamFormatCMAF)
	}
}

func TestListStreamsResponse_EmptyList(t *testing.T) {
	raw := `{"streams":[]}`
	var resp ListStreamsResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		t.Fatalf("json.Unmarshal() error: %v", err)
	}
	if len(resp.Streams) != 0 {
		t.Errorf("len(Streams) = %d, want 0", len(resp.Streams))
	}
}
