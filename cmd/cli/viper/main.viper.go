package main

import (
	"fmt"

	"github.com/spf13/viper"
)

// Trường hợp nhiều database thì tạo struct để lưu trữ
type Config struct {
	Server struct {
		Port int `mapstructure:"port"`
	} `mapstructure:"server"`
	Databases []struct {
		User     string `mapstructure:"user"`
		Password string `mapstructure:"password"`
		Host     string `mapstructure:"host"`
	} `mapstructure:"databases"`
}

func main() {
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

	var config Config

	// hàm Unmarshal để đọc đối tượng phức tạp
	if err := viper.Unmarshal(&config); err != nil {
		fmt.Println("Unable to decode configuration")
	}

	fmt.Println("Config Server Port: ", config.Server.Port)
	// fmt.Println("Config Database Name: ", config.Databases)

	for _, db := range config.Databases {
		fmt.Printf("User: %s, Password: %s, Host: %s\n", db.User, db.Password, db.Host)
	}
}
