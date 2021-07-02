package httpfileserver

import "testing"

func Test_fileSizeBytes_String(t *testing.T) {
	tests := []struct {
		name string
		f    fileSizeBytes
		want string
	}{
		{
			name: "bytes",
			f:    123,
			want: "123",
		},
		{
			name: "KB",
			f:    1234,
			want: "1K",
		},
		{
			name: "MB",
			f:    1234567,
			want: "1M",
		},
		{
			name: "G",
			f:    1234567890,
			want: "1G",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.f.String(); got != tt.want {
				t.Errorf("fileSizeBytes.String() = %v, want %v", got, tt.want)
			}
		})
	}
}
