package exec

import (
	"fmt"
	"reflect"
	"testing"
)

// Test entrypoint string creation for exec command

func TestGenerateEntrypoint(t *testing.T) {
	// Define test cases
	tests := []struct {
		name      string
		runID     string
		targetPID int
		cmd       []string
		wantErr   bool
	}{
		{
			name:      "Basic functionality",
			runID:     "as5asd5",
			targetPID: 12345,
			cmd:       []string{"ls", "-l"},
			wantErr:   false,
		},
		{
			name:      "Empty command",
			runID:     "jkgkgr3",
			targetPID: 12345,
			cmd:       []string{},
			wantErr:   false,
		},
		{
			name:      "Command with special characters",
			runID:     "testRunID",
			targetPID: 12345,
			cmd:       []string{"ls", "-l", "|", "grep", "test"},
			wantErr:   false,
		},
	}

	// Run test cases
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Call function
			got := generateEntrypoint(tt.runID, tt.targetPID, tt.cmd, true, []string{})
			fmt.Printf("got: %s\n", got)

			// Check for panic
			if r := recover(); r != nil && !tt.wantErr {
				t.Errorf("generateEntrypoint() panic = %v, wantErr %v", r, tt.wantErr)
			}

			// Check result
			if len(got) == 0 && !tt.wantErr {
				t.Errorf("generateEntrypoint() = %v, want non-empty string", got)
			}
		})
	}
}

func TestShellescape(t *testing.T) {
	// Define test cases
	tests := []struct {
		name string
		args []string
		want []string
	}{
		{
			name: "No arguments",
			args: []string{},
			want: []string{},
		},
		{
			name: "Single argument without spaces",
			args: []string{"arg"},
			want: []string{"\"arg\""},
		},
		{
			name: "Single argument with spaces",
			args: []string{"arg with spaces"},
			want: []string{"\"arg with spaces\""},
		},
		{
			name: "Multiple arguments",
			args: []string{"arg1", "arg2 \n space", "arg3"},
			want: []string{"\"arg1\"", "\"arg2 space\"", "\"arg3\""},
		},
		{
			name: "Arguments with special characters",
			args: []string{"arg1$", "arg2&", "arg3*"},
			want: []string{"\"arg1$\"", "\"arg2&\"", "\"arg3*\""},
		},
	}

	// Run test cases
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := shellescape(tt.args)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("shellescape() = %v, want %v", got, tt.want)
			}
		})
	}
}
