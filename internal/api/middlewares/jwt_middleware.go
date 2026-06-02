package middlewares

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"restapi/pkg/utils"

	"github.com/golang-jwt/jwt/v5"
)

func JWTMiddleware(next http.Handler) http.Handler {
	fmt.Println("--------JWT Middleware--------")
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("-------- Inside JWT Middleware--------")
		token, err := r.Cookie("Bearer")
		if err != nil {
			http.Error(w, "Authorization Header Missing", http.StatusUnauthorized)
			return
		}

		jwtSecret := os.Getenv("JWT_SECRET")
		parsedToken, err := jwt.Parse(token.Value, func(token *jwt.Token) (any, error) {
			// hmacSampleSecret is a []byte containing your secret, e.g. []byte("my_secret_key")
			return []byte(jwtSecret), nil
		}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
		if err != nil {
			if errors.Is(err, jwt.ErrTokenExpired) {
				http.Error(w, "Token Expired", http.StatusUnauthorized)
				return
			}
			utils.ErrorHandler(err, "")
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}

		if parsedToken.Valid {
			log.Println("Valid JWT")
		} else {
			http.Error(w, "Invalid JWT token", http.StatusUnauthorized)
			return
		}
		claims, ok := parsedToken.Claims.(jwt.MapClaims)
		if !ok {
			log.Println(err)
			http.Error(w, "Error while checking jwt token", http.StatusInternalServerError)
			return
		}

		ctx := context.WithValue(r.Context(), "role", claims["role"])
		ctx = context.WithValue(ctx, "username", claims["username"])
		ctx = context.WithValue(ctx, "expiresAt", claims["exp"])
		ctx = context.WithValue(ctx, "userID", claims["uid"])

		fmt.Println(ctx)
		next.ServeHTTP(w, r.WithContext(ctx))
		fmt.Println("--------The end of JWT Middleware--------")

	})
}
