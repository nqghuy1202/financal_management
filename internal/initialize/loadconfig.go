package initialize

import (
	"financal_management/global"
	"fmt"

	"github.com/spf13/viper"
)

func LoadConfig() {
	viper := viper.New()
	viper.AddConfigPath("./config/") // path to config
	viper.SetConfigName("local")     // file name
	viper.SetConfigType("yaml")      // file type

	// read configuration file
	err := viper.ReadInConfig()
	if err != nil {
		// panic: stop program
		panic(fmt.Errorf("Failed to read configuaration %w\n", err))
	}

	// success read configuration
	fmt.Println("Server Port: ", viper.GetInt("server.port"))
	fmt.Println("Security Key: ", viper.GetString("security.jwt.key"))

	// hàm Unmarshal để đọc đối tượng phức tạp
	if err := viper.Unmarshal(&global.Config); err != nil {
		fmt.Println("Unable to decode configuration")
	}
}
