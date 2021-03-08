package validate

import (
	"log"
	"testing"
)

func TestValidSnowflake(t *testing.T) {
	snowflake := ""
	v, err := ValidSnowflake(snowflake)
	if v || err == nil {
		t.Fatal("expected empty snowflake to be invalid and/or return non-nil error")
	}

	snowflake = "-1"
	v, err = ValidSnowflake(snowflake)
	if v || err == nil {
		t.Fatal("expected snowflake=-1 to be invalid and/or return non-nil error")
	}

	snowflake = "1000"
	v, err = ValidSnowflake(snowflake)
	if v || err == nil {
		t.Fatal("expected snowflake=1000 to be invalid and/or return non-nil error")
	}

	snowflake = "abcd123"
	v, err = ValidSnowflake(snowflake)
	if v || err == nil {
		log.Println(err)
		t.Fatal("expected snowflake=1000 to be invalid and/or return non-nil error")
	}

	snowflake = "754465589958803548"
	v, err = ValidSnowflake(snowflake)
	if !v || err != nil {
		t.Fatal("expected snowflake=754465589958803548 to be valid and return nil error")
	}
}
