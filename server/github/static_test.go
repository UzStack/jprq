package github

import "testing"

func TestStaticAuthenticator(t *testing.T) {
	auth := NewStatic("correct-token")
	if _, err := auth.Authenticate("wrong-token"); err == nil {
		t.Fatal("wrong token was accepted")
	}
	user, err := auth.Authenticate("correct-token")
	if err != nil {
		t.Fatal(err)
	}
	if !user.Allowed || user.Login != "selfhost" {
		t.Fatalf("unexpected user: %+v", user)
	}
}
