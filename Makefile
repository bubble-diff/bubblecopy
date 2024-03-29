# DEBUG_SETTINGS is used for local test easily.
define DEBUG_SETTINGS
{
	"taskid": 2,
	"interface": "lo0",
	"service_port": "8080",
	"replay_svr_addr": "127.0.0.1:6789"
}
endef
export DEBUG_SETTINGS

# todo: cross-compile for linux and windows to releases.
build: clean
	@echo "go build project..."
	@mkdir -p output
	@go build -o output/bubblecopy

run: build
	@cd output && sudo ./bubblecopy

run-debug: build
	@echo "loading debug settings..."
	@touch output/settings.json
	@echo "$$DEBUG_SETTINGS" > output/settings.json
	@cd output && sudo ./bubblecopy -debug

clean:
	@echo "clean output directory..."
	@rm -rf output/

update-idl:
	@rm -rf ./idl
	@echo "step1> fetching idl repo..."
	@git clone --depth=1 https://github.com/bubble-diff/IDL.git idl
	@rm -rf ./idl/.git
	@rm ./idl/.gitignore
	@echo "step2> compile idl..."
	@protoc --go_out=. idl/*.proto
	@go mod tidy
