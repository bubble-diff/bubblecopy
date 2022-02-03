# DEBUG_SETTINGS is used for local test easily.
define DEBUG_SETTINGS
{
	"interface": "lo0",
	"service_port": "8080"
}
endef
export DEBUG_SETTINGS

# todo: cross-compile for linux and windows to releases.
build: clean
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