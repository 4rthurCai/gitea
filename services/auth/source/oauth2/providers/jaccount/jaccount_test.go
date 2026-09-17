package jaccount

import (
	"strings"
	"testing"

	"github.com/markbates/goth"
	"github.com/stretchr/testify/require"
)

func TestUserFromReaderPreservesJAccountIdentity(t *testing.T) {
	var user goth.User
	err := userFromReader(strings.NewReader(`{"entities":[{"name":"张三","account":"student","id":"account-id","code":"123456"}]}`), &user)
	require.NoError(t, err)
	require.Equal(t, "张三", user.Name)
	require.Equal(t, "student", user.NickName)
	require.Equal(t, "student@sjtu.edu.cn", user.Email)
	require.Equal(t, "account-id", user.UserID)
	require.Equal(t, "123456", user.Location)
}
