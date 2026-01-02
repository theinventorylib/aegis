module example-basic-auth

go 1.25.5

require (
	github.com/go-chi/chi/v5 v5.2.2
	github.com/lib/pq v1.10.9
	github.com/theinventorylib/aegis v0.1.0
)

require (
	github.com/asaskevich/govalidator v0.0.0-20200108200545-475eaeb16496 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/go-ozzo/ozzo-validation/v4 v4.3.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/oklog/ulid/v2 v2.1.1 // indirect
	github.com/redis/go-redis/v9 v9.17.2 // indirect
	golang.org/x/crypto v0.46.0 // indirect
	golang.org/x/sys v0.39.0 // indirect
)

// For local development, using local aegis code
replace github.com/theinventorylib/aegis => ../../
