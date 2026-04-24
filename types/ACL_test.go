package types

import "testing"

func TestConvertToPermission(t *testing.T) {
	tests := []struct {
		input string
		want  Permission
	}{
		{"read", READ},
		{"READ", READ},
		{"Read", READ},
		{"write", WRITE},
		{"WRITE", WRITE},
		{"read-acp", READ_ACP},
		{"write-acp", WRITE_ACP},
		{"full-control", FULL_CONTROL},
		{"FULL-CONTROL", FULL_CONTROL},
		{"garbage", ACLUnknown},
		{"", ACLUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := ConvertToPermission(tt.input)
			if got != tt.want {
				t.Errorf("ConvertToPermission(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestConvertToBucketACL(t *testing.T) {
	tests := []struct {
		input string
		want  Permission
	}{
		{"private", BUCKET_ACLPrivate},
		{"public-read", BUCKET_ACLPublicRead},
		{"public-read-write", BUCKET_ACLPublicReadWrite},
		{"public-write", BUCKET_ACLPublicWrite},
		{"garbage", ACLUnknown},
		{"PRIVATE", ACLUnknown}, // case-sensitive
		{"", ACLUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := ConvertToBucketACL(tt.input)
			if got != tt.want {
				t.Errorf("ConvertToBucketACL(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestIsWritePermission(t *testing.T) {
	tests := []struct {
		perm Permission
		want bool
	}{
		{WRITE, true},
		{WRITE_ACP, true},
		{FULL_CONTROL, true},
		{READ, false},
		{READ_ACP, false},
		{ACLUnknown, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.perm), func(t *testing.T) {
			got := IsWritePermission(tt.perm)
			if got != tt.want {
				t.Errorf("IsWritePermission(%q) = %v, want %v", tt.perm, got, tt.want)
			}
		})
	}
}

func TestIsReadPermission(t *testing.T) {
	tests := []struct {
		perm Permission
		want bool
	}{
		{READ, true},
		{READ_ACP, true},
		{FULL_CONTROL, true},
		{WRITE, false},
		{WRITE_ACP, false},
		{ACLUnknown, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.perm), func(t *testing.T) {
			got := IsReadPermission(tt.perm)
			if got != tt.want {
				t.Errorf("IsReadPermission(%q) = %v, want %v", tt.perm, got, tt.want)
			}
		})
	}
}

func TestIsBucketACL(t *testing.T) {
	tests := []struct {
		perm Permission
		want bool
	}{
		{BUCKET_ACLPrivate, true},
		{BUCKET_ACLPublicRead, true},
		{BUCKET_ACLPublicWrite, true},
		{BUCKET_ACLPublicReadWrite, true},
		{READ, false},
		{FULL_CONTROL, false},
		{ACLUnknown, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.perm), func(t *testing.T) {
			got := IsBucketACL(tt.perm)
			if got != tt.want {
				t.Errorf("IsBucketACL(%q) = %v, want %v", tt.perm, got, tt.want)
			}
		})
	}
}

func TestAWSHeaderToACL(t *testing.T) {
	tests := []struct {
		header string
		want   Permission
	}{
		{"x-amz-grant-read", READ},
		{"x-amz-grant-write", WRITE},
		{"x-amz-grant-read-acp", READ_ACP},
		{"x-amz-grant-write-acp", WRITE_ACP},
		{"x-amz-grant-full-control", FULL_CONTROL},
		{"x-amz-unknown", ACLUnknown},
		{"", ACLUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.header, func(t *testing.T) {
			got := AWSHeaderToACL(tt.header)
			if got != tt.want {
				t.Errorf("AWSHeaderToACL(%q) = %q, want %q", tt.header, got, tt.want)
			}
		})
	}
}

func TestDescribe(t *testing.T) {
	// Known permission+resource pair
	desc, ok := Describe(READ, ResourceBucket)
	if !ok {
		t.Fatal("expected Describe(READ, ResourceBucket) to return true")
	}
	if desc == "" {
		t.Fatal("expected non-empty description")
	}

	// Unknown pair
	_, ok = Describe(ACLUnknown, ResourceBucket)
	if ok {
		t.Fatal("expected Describe(ACLUnknown, ResourceBucket) to return false")
	}
}

func TestIsBucketACLReadWrite(t *testing.T) {
	if !IsBucketACLRead(BUCKET_ACLPublicRead) {
		t.Error("expected IsBucketACLRead(PUBLIC_READ) = true")
	}
	if !IsBucketACLRead(BUCKET_ACLPublicReadWrite) {
		t.Error("expected IsBucketACLRead(PUBLIC_READ_WRITE) = true")
	}
	if IsBucketACLRead(BUCKET_ACLPrivate) {
		t.Error("expected IsBucketACLRead(PRIVATE) = false")
	}

	if !IsBucketACLWrite(BUCKET_ACLPublicWrite) {
		t.Error("expected IsBucketACLWrite(PUBLIC_WRITE) = true")
	}
	if !IsBucketACLWrite(BUCKET_ACLPublicReadWrite) {
		t.Error("expected IsBucketACLWrite(PUBLIC_READ_WRITE) = true")
	}
	if IsBucketACLWrite(BUCKET_ACLPrivate) {
		t.Error("expected IsBucketACLWrite(PRIVATE) = false")
	}
}
