module github.com/garyjdn/go-rustfs

go 1.24.0

require (
	github.com/garyjdn/go-apperror v1.0.1
	github.com/garyjdn/go-auditlogger v1.0.0
	github.com/google/uuid v1.6.0
)

// Local development dependencies
replace github.com/garyjdn/go-apperror => ../../shared/app-error
replace github.com/garyjdn/go-auditlogger => ../../shared/audit