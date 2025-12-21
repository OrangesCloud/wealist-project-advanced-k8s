// Package health는 K8s 헬스체크 엔드포인트를 제공합니다.
// 이 파일은 헬스체크에서 사용하는 인터페이스를 정의합니다.
package health

import "context"

// Pinger는 연결 상태를 확인하기 위한 인터페이스입니다.
// DB, Redis 등의 의존성을 추상화하여 테스트에서 mock할 수 있습니다.
//
// 사용 예:
//
//	// Mock 구현
//	type mockPinger struct {
//	    err error
//	}
//	func (m *mockPinger) Ping(ctx context.Context) error { return m.err }
//
//	// 테스트에서 사용
//	checker := health.NewHealthCheckerWithPinger(&mockPinger{}, nil)
type Pinger interface {
	// Ping은 연결 상태를 확인합니다.
	// 연결이 정상이면 nil을 반환하고, 그렇지 않으면 에러를 반환합니다.
	Ping(ctx context.Context) error
}

// DBPinger는 *gorm.DB를 Pinger 인터페이스로 래핑합니다.
// gorm.DB의 Ping 메서드를 호출합니다.
type DBPinger struct {
	db interface {
		DB() (interface {
			PingContext(ctx context.Context) error
		}, error)
	}
}

// NewDBPinger는 gorm.DB를 감싸는 DBPinger를 생성합니다.
func NewDBPinger(db interface {
	DB() (interface {
		PingContext(ctx context.Context) error
	}, error)
}) *DBPinger {
	return &DBPinger{db: db}
}

// Ping은 DB 연결 상태를 확인합니다.
func (d *DBPinger) Ping(ctx context.Context) error {
	if d.db == nil {
		return nil
	}
	sqlDB, err := d.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.PingContext(ctx)
}

// RedisPinger는 *redis.Client를 Pinger 인터페이스로 래핑합니다.
type RedisPinger struct {
	redis interface {
		Ping(ctx context.Context) interface {
			Err() error
		}
	}
}

// NewRedisPinger는 redis.Client를 감싸는 RedisPinger를 생성합니다.
func NewRedisPinger(redis interface {
	Ping(ctx context.Context) interface {
		Err() error
	}
}) *RedisPinger {
	return &RedisPinger{redis: redis}
}

// Ping은 Redis 연결 상태를 확인합니다.
func (r *RedisPinger) Ping(ctx context.Context) error {
	if r.redis == nil {
		return nil
	}
	return r.redis.Ping(ctx).Err()
}

// MockPinger는 테스트용 Pinger 구현체입니다.
type MockPinger struct {
	// Err는 Ping 호출 시 반환할 에러입니다.
	Err error
}

// Ping은 설정된 에러를 반환합니다.
func (m *MockPinger) Ping(ctx context.Context) error {
	return m.Err
}
