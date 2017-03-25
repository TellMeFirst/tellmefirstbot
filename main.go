package main

import (
    "github.com/AntonioLangiu/tellmefirstbot/bot"
    "github.com/AntonioLangiu/tellmefirstbot/common"
)

func main() {
    configuration := common.LoadConfiguration()
    bot.LoadBot(configuration)
}
