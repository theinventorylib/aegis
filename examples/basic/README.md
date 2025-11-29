# Basic Aegis Example

This example demonstrates how to use Aegis authentication framework with Chi router.

## Setup

1. Install dependencies:
```bash
go mod download
```

2. Set up PostgreSQL database:
```bash
createdb aegis_db
```

3. Update the connection string in `main.go` with your database credentials.

4. Run the example:
```bash
go run main.go
```

The server will start on `http://localhost:3000`.

## API Endpoints

### Authentication

- `POST /auth/signup` - Create a new account
  ```json
  {
    "email": "user@example.com",
    "password": "<your-password>"
  }
  ```

- `POST /auth/login` - Login with credentials
  ```json
  {
    "email": "user@example.com",
    "password": "<your-password>"
  }
  ```

- `POST /auth/logout` - Logout and invalidate session

- `GET /auth/user` - Get current authenticated user

### OTP

- `POST /auth/otp/send` - Send OTP for verification
  ```json
  {
    "email": "user@example.com",
    "purpose": "email_verification"
  }
  ```

- `POST /auth/otp/verify` - Verify OTP code
  ```json
  {
    "email": "user@example.com",
    "code": "123456",
    "purpose": "email_verification"
  }
  ```

### Password Reset

- `POST /auth/password/reset/request` - Request password reset
  ```json
  {
    "email": "user@example.com"
  }
  ```

- `POST /auth/password/reset/confirm` - Confirm password reset
  ```json
  {
    "email": "user@example.com",
    "code": "123456",
    "newPassword": "newsecurepassword"
  }
  ```

### Protected Routes

- `GET /api/protected` - Example protected endpoint (requires authentication)

## Testing

### 1. Signup
```bash
curl -X POST http://localhost:3000/auth/signup \
  -H "Content-Type: application/json" \
    -d '{"email":"test@example.com","password":"<your-password>"}'
```

### 2. Login
```bash
curl -X POST http://localhost:3000/auth/login \
  -H "Content-Type: application/json" \
    -d '{"email":"test@example.com","password":"<your-password>"}' \
  -c cookies.txt
```

### 3. Access Protected Route
```bash
curl -X GET http://localhost:3000/api/protected \
  -b cookies.txt
```

### 4. Logout
```bash
curl -X POST http://localhost:3000/auth/logout \
  -b cookies.txt
```
