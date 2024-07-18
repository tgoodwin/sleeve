package client

import "testing"

func Test_createFixedLengthHash(t *testing.T) {
	type args struct {
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Test createFixedLengthHash",
			args: args{},
			want: "f4e3d2c1b0a9",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := createFixedLengthHash(); got != tt.want {
				t.Errorf("createFixedLengthHash() = %v, want %v", got, tt.want)
			}
		})
	}
}
