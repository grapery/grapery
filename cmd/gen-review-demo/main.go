// Command gen-review-demo prints bcrypt hash and VipPay FNV user_id for the App Review demo account.
//
// Usage:
//
//	go run ./cmd/gen-review-demo
//	go run ./cmd/gen-review-demo -password 'Gr@p3ryIap2026!'
package main

import (
	"flag"
	"fmt"
	"hash/fnv"
	"os"

	authPkg "github.com/grapestree/fgrapery/grapery/internal/auth"
)

const defaultReviewUserID = "revw-jingjing-0000-8000-000000000001"

func main() {
	password := flag.String("password", "Gr@p3ryIap2026!", "demo account password to hash")
	userID := flag.String("user-id", defaultReviewUserID, "Grapery user UUID for VipPay FNV mapping")
	flag.Parse()

	hash, err := authPkg.HashPassword(*password)
	if err != nil {
		fmt.Fprintf(os.Stderr, "hash password: %v\n", err)
		os.Exit(1)
	}

	fnvID := stringToInt64(*userID)

	fmt.Printf("Review demo identifiers\n")
	fmt.Printf("  user_uuid:        %s\n", *userID)
	fmt.Printf("  password:         %s\n", *password)
	fmt.Printf("  password_hash:    %s\n", hash)
	fmt.Printf("  vippay_user_id:   %d\n", fnvID)
	fmt.Printf("  vippay_user_id_u: %d\n", uint64(fnvID))
}

func stringToInt64(s string) int64 {
	if s == "" {
		return 0
	}
	h := fnv.New64a()
	_, _ = h.Write([]byte(s))
	return int64(h.Sum64())
}
