package initialize

func Run() {
	// load configuration

	LoadConfig()
	InitLogger()
	InitMySQL()
	InitRedis()

	r := InitRouter()

	r.Run(":8002")
}
