package main

import (
	"flag"
	"os"
	"os/signal"
	"syscall"

	mcp "github.com/metoro-io/mcp-golang"
	"github.com/metoro-io/mcp-golang/transport/http"
	log "github.com/sirupsen/logrus"

	"github.com/grapery/grapery/config"
	"github.com/grapery/grapery/service/mcps"
	"github.com/grapery/grapery/version"
)

var printVersion = flag.Bool("version", false, "app build version")
var configPath = flag.String("config", "config.json", "config file")
var serverAddr = flag.String("addr", ":8080", "server address")

func main() {
	flag.Parse()
	if *printVersion {
		version.PrintFullVersionInfo()
		return
	}

	err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatal("read config failed : ", err)
	}

	err = config.ValiedConfig(config.GlobalConfig)
	if err != nil {
		log.Fatal("Validate config failed : ", err)
	}

	// Create HTTP transport
	transport := http.NewHTTPTransport("/mcp")
	transport.WithAddr(*serverAddr)

	// Create MCP server
	server := mcp.NewServer(transport)

	// Create and initialize MCP service
	service := mcps.NewMcpService()
	err = service.Initialize(config.GlobalConfig)
	if err != nil {
		log.Fatal("initialize service failed : ", err)
	}

	// Register tools
	err = registerTools(server, service)
	if err != nil {
		log.Fatal("register tools failed : ", err)
	}

	// Register prompts
	err = registerPrompts(server, service)
	if err != nil {
		log.Fatal("register prompts failed : ", err)
	}

	// Register resources
	err = registerResources(server, service)
	if err != nil {
		log.Fatal("register resources failed : ", err)
	}

	// Start server
	go func() {
		if err := server.Serve(); err != nil {
			log.Fatal("start server failed : ", err)
		}
	}()

	// Handle shutdown signals
	sc := make(chan os.Signal, 1)
	signal.Notify(sc,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
	)

	select {
	case s := <-sc:
		log.Info("Received signal: ", s.String())
		if err := service.Shutdown(); err != nil {
			log.Error("Error shutting down service: ", err)
		}
	}
}

func registerTools(server *mcp.Server, service *mcps.McpService) error {
	// Register story management tools
	err := server.RegisterTool("create_story", "create_story", &mcps.CreateStoryTool{
		BaseTool: mcps.BaseTool{
			Service: service,
		},
	})
	if err != nil {
		return err
	}

	err = server.RegisterTool("get_story", "get_story", &mcps.GetStoryTool{
		BaseTool: mcps.BaseTool{
			Service: service,
		},
	})
	if err != nil {
		return err
	}

	// Register character management tools
	err = server.RegisterTool("create_character", "create_character", &mcps.CreateCharacterTool{
		BaseTool: mcps.BaseTool{
			Service: service,
		},
	})
	if err != nil {
		return err
	}

	err = server.RegisterTool("get_character", "get_character", &mcps.GetCharacterTool{
		BaseTool: mcps.BaseTool{
			Service: service,
		},
	})
	if err != nil {
		return err
	}

	// Register user interaction tools
	err = server.RegisterTool("follow_character", "follow_character", &mcps.FollowCharacterTool{
		BaseTool: mcps.BaseTool{
			Service: service,
		},
	})
	if err != nil {
		return err
	}

	err = server.RegisterTool("unfollow_character", "unfollow_character", &mcps.UnfollowCharacterTool{
		BaseTool: mcps.BaseTool{
			Service: service,
		},
	})
	if err != nil {
		return err
	}

	err = server.RegisterTool("like_story", "like_story", &mcps.LikeStoryTool{
		BaseTool: mcps.BaseTool{
			Service: service,
		},
	})
	if err != nil {
		return err
	}

	err = server.RegisterTool("unlike_story", "unlike_story", &mcps.UnlikeStoryTool{
		BaseTool: mcps.BaseTool{
			Service: service,
		},
	})
	if err != nil {
		return err
	}

	return nil
}

func registerPrompts(server *mcp.Server, service *mcps.McpService) error {
	// Register story generation prompts
	err := server.RegisterPrompt("generate_story", "Generate a new story", &mcps.GenerateStoryPrompt{
		BasePrompt: mcps.BasePrompt{
			Service: service,
		},
	})
	if err != nil {
		return err
	}

	// Register character generation prompts
	err = server.RegisterPrompt("generate_character", "Generate a new character", &mcps.GenerateCharacterPrompt{
		BasePrompt: mcps.BasePrompt{
			Service: service,
		},
	})
	if err != nil {
		return err
	}

	return nil
}

func registerResources(server *mcp.Server, service *mcps.McpService) error {
	// Register story resources
	err := server.RegisterResource("story://", "story_resource", "Story resource", "application/json", &mcps.StoryResource{
		BaseResource: mcps.BaseResource{
			Service: service,
		},
	})
	if err != nil {
		return err
	}

	// Register character resources
	err = server.RegisterResource("character://", "character_resource", "Character resource", "application/json", &mcps.CharacterResource{
		BaseResource: mcps.BaseResource{
			Service: service,
		},
	})
	if err != nil {
		return err
	}

	return nil
}
