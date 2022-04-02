// Copyright 2020 tree xie
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package util

import (
	"context"
)

type contextKey string

const (
	accountKey contextKey = "account"
	traceIDKey contextKey = "traceID"
)

func getStringFromContext(ctx context.Context, key contextKey) string {
	v := ctx.Value(key)
	if v == nil {
		return ""
	}
	s, _ := v.(string)
	return s
}

// SetAccount sets account to context
func SetAccount(ctx context.Context, account string) context.Context {
	return context.WithValue(ctx, accountKey, account)
}

// GetAccount gets account from context
func GetAccount(ctx context.Context) string {
	return getStringFromContext(ctx, accountKey)
}

// SetTraceID sets trace id to context
func SetTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, traceIDKey, traceID)
}

// GetTraceID gets trace id from context
func GetTraceID(ctx context.Context) string {
	return getStringFromContext(ctx, traceIDKey)
}
