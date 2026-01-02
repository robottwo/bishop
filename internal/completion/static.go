package completion

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/robottwo/bishop/pkg/shellinput"
	"gopkg.in/yaml.v3"
)

// StaticCompleter handles static word lists for common commands
type StaticCompleter struct {
	completions map[string][]shellinput.CompletionCandidate
	mu          sync.RWMutex
}

// UserCompletionConfig represents user-defined completion configuration
type UserCompletionConfig struct {
	Commands map[string][]UserCompletion `yaml:"commands" json:"commands"`
}

// UserCompletion represents a single user-defined completion entry
type UserCompletion struct {
	Value       string `yaml:"value" json:"value"`
	Description string `yaml:"description,omitempty" json:"description,omitempty"`
}

func NewStaticCompleter() *StaticCompleter {
	sc := &StaticCompleter{
		completions: make(map[string][]shellinput.CompletionCandidate),
	}
	sc.registerDefaults()
	sc.loadUserCompletions()
	return sc
}

func (s *StaticCompleter) registerDefaults() {
	// Docker - container management
	s.registerWithDesc("docker", []subCmd{
		{"attach", "Attach local stdin/stdout/stderr to a running container"},
		{"build", "Build an image from a Dockerfile"},
		{"commit", "Create a new image from a container's changes"},
		{"compose", "Docker Compose commands"},
		{"container", "Manage containers"},
		{"cp", "Copy files between container and local filesystem"},
		{"create", "Create a new container"},
		{"diff", "Inspect changes to files on a container's filesystem"},
		{"events", "Get real-time events from the server"},
		{"exec", "Execute a command in a running container"},
		{"export", "Export a container's filesystem as a tar archive"},
		{"history", "Show the history of an image"},
		{"image", "Manage images"},
		{"images", "List images"},
		{"import", "Import from a tarball to create a filesystem image"},
		{"info", "Display system-wide information"},
		{"inspect", "Return low-level information on Docker objects"},
		{"kill", "Kill one or more running containers"},
		{"load", "Load an image from a tar archive"},
		{"login", "Log in to a registry"},
		{"logout", "Log out from a registry"},
		{"logs", "Fetch the logs of a container"},
		{"network", "Manage networks"},
		{"pause", "Pause all processes within a container"},
		{"plugin", "Manage plugins"},
		{"port", "List port mappings for a container"},
		{"ps", "List containers"},
		{"pull", "Pull an image from a registry"},
		{"push", "Push an image to a registry"},
		{"rename", "Rename a container"},
		{"restart", "Restart one or more containers"},
		{"rm", "Remove one or more containers"},
		{"rmi", "Remove one or more images"},
		{"run", "Run a command in a new container"},
		{"save", "Save images to a tar archive"},
		{"search", "Search Docker Hub for images"},
		{"start", "Start one or more stopped containers"},
		{"stats", "Display live container resource usage statistics"},
		{"stop", "Stop one or more running containers"},
		{"system", "Manage Docker"},
		{"tag", "Create a tag to reference an image"},
		{"top", "Display running processes of a container"},
		{"unpause", "Unpause all processes within a container"},
		{"update", "Update configuration of containers"},
		{"version", "Show Docker version information"},
		{"volume", "Manage volumes"},
		{"wait", "Block until containers stop, then print exit codes"},
	})

	// Docker Compose
	s.registerWithDesc("docker-compose", []subCmd{
		{"build", "Build or rebuild services"},
		{"config", "Validate and view the Compose file"},
		{"create", "Create services"},
		{"down", "Stop and remove containers, networks"},
		{"events", "Receive real-time events from containers"},
		{"exec", "Execute a command in a running container"},
		{"images", "List images"},
		{"kill", "Kill containers"},
		{"logs", "View output from containers"},
		{"pause", "Pause services"},
		{"port", "Print the public port for a port binding"},
		{"ps", "List containers"},
		{"pull", "Pull service images"},
		{"push", "Push service images"},
		{"restart", "Restart services"},
		{"rm", "Remove stopped containers"},
		{"run", "Run a one-off command"},
		{"scale", "Set number of containers for a service"},
		{"start", "Start services"},
		{"stop", "Stop services"},
		{"top", "Display running processes"},
		{"unpause", "Unpause services"},
		{"up", "Create and start containers"},
		{"version", "Show version information"},
	})

	// Kubernetes kubectl
	s.registerWithDesc("kubectl", []subCmd{
		{"annotate", "Update annotations on a resource"},
		{"api-resources", "Print the supported API resources"},
		{"api-versions", "Print the supported API versions"},
		{"apply", "Apply a configuration to a resource by file or stdin"},
		{"attach", "Attach to a running container"},
		{"auth", "Inspect authorization"},
		{"autoscale", "Auto-scale a deployment, replicaset, or statefulset"},
		{"certificate", "Modify certificate resources"},
		{"cluster-info", "Display cluster information"},
		{"completion", "Output shell completion code"},
		{"config", "Modify kubeconfig files"},
		{"cordon", "Mark node as unschedulable"},
		{"cp", "Copy files to and from containers"},
		{"create", "Create a resource from a file or stdin"},
		{"debug", "Create debugging sessions for workloads"},
		{"delete", "Delete resources"},
		{"describe", "Show details of a resource"},
		{"diff", "Diff the live version against a config"},
		{"drain", "Drain node for maintenance"},
		{"edit", "Edit a resource"},
		{"events", "List events"},
		{"exec", "Execute a command in a container"},
		{"explain", "Get documentation for a resource"},
		{"expose", "Expose a resource as a new service"},
		{"get", "Display one or many resources"},
		{"kustomize", "Build a kustomization target"},
		{"label", "Update labels on a resource"},
		{"logs", "Print logs for a container"},
		{"patch", "Update fields of a resource"},
		{"plugin", "Manage plugins"},
		{"port-forward", "Forward local ports to a pod"},
		{"proxy", "Run a proxy to the Kubernetes API server"},
		{"replace", "Replace a resource"},
		{"rollout", "Manage rollouts of a resource"},
		{"run", "Run a particular image on the cluster"},
		{"scale", "Set a new size for a deployment"},
		{"set", "Set specific features on objects"},
		{"taint", "Update taints on nodes"},
		{"top", "Display resource usage (CPU/memory)"},
		{"uncordon", "Mark node as schedulable"},
		{"version", "Print client and server version"},
		{"wait", "Wait for a condition on resources"},
	})

	// npm - Node.js package manager
	s.registerWithDesc("npm", []subCmd{
		{"access", "Set access level on published packages"},
		{"adduser", "Add a registry user account"},
		{"audit", "Run a security audit"},
		{"bin", "Display npm bin folder"},
		{"bugs", "Report bugs for a package"},
		{"cache", "Manipulate packages cache"},
		{"ci", "Install project with clean slate"},
		{"completion", "Tab completion for npm"},
		{"config", "Manage npm configuration"},
		{"dedupe", "Reduce duplication in package tree"},
		{"deprecate", "Deprecate a version of a package"},
		{"diff", "Show changes between package versions"},
		{"dist-tag", "Modify package distribution tags"},
		{"docs", "Open docs for a package in a web browser"},
		{"doctor", "Check your npm environment"},
		{"edit", "Edit an installed package"},
		{"exec", "Run a command from a local or remote npm package"},
		{"explain", "Explain installed packages"},
		{"explore", "Browse an installed package"},
		{"find-dupes", "Find duplicate packages"},
		{"fund", "Display funding information"},
		{"help", "Get help on npm"},
		{"hook", "Manage registry hooks"},
		{"init", "Create a package.json file"},
		{"install", "Install a package"},
		{"install-ci-test", "Install and run tests"},
		{"install-test", "Install and test a package"},
		{"link", "Create symlink to a package folder"},
		{"login", "Log in to a registry"},
		{"logout", "Log out of a registry"},
		{"ls", "List installed packages"},
		{"outdated", "Check for outdated packages"},
		{"owner", "Manage package owners"},
		{"pack", "Create a tarball from a package"},
		{"ping", "Ping npm registry"},
		{"pkg", "Manage package.json"},
		{"prefix", "Display prefix"},
		{"profile", "Manage registry profile"},
		{"prune", "Remove extraneous packages"},
		{"publish", "Publish a package"},
		{"rebuild", "Rebuild a package"},
		{"repo", "Open package repository in browser"},
		{"restart", "Restart a package"},
		{"root", "Display npm root"},
		{"run", "Run arbitrary package scripts"},
		{"search", "Search for packages"},
		{"set", "Set a config key"},
		{"shrinkwrap", "Lock down dependency versions"},
		{"star", "Mark favorite packages"},
		{"stars", "View starred packages"},
		{"start", "Start a package"},
		{"stop", "Stop a package"},
		{"team", "Manage organization teams"},
		{"test", "Test a package"},
		{"token", "Manage authentication tokens"},
		{"uninstall", "Remove a package"},
		{"unpublish", "Remove a package from registry"},
		{"unstar", "Remove package from starred list"},
		{"update", "Update packages"},
		{"version", "Manage package version"},
		{"view", "View registry info"},
		{"whoami", "Display npm username"},
	})

	// yarn - JavaScript package manager
	s.registerWithDesc("yarn", []subCmd{
		{"add", "Install a package"},
		{"audit", "Run vulnerability audit"},
		{"autoclean", "Clean and remove unnecessary files"},
		{"bin", "Display the folder for yarn binaries"},
		{"cache", "Manage the global cache"},
		{"check", "Verify package dependencies"},
		{"config", "Manage the yarn configuration files"},
		{"create", "Create new projects from create-* starter kits"},
		{"dedupe", "Deduplicate dependencies"},
		{"dlx", "Run a package in a temporary environment"},
		{"exec", "Execute a shell command"},
		{"generate-lock-entry", "Generate a lock file entry"},
		{"global", "Install packages globally"},
		{"help", "Display help information"},
		{"import", "Generate yarn.lock from package-lock.json"},
		{"info", "Show information about a package"},
		{"init", "Create a new package.json"},
		{"install", "Install all dependencies"},
		{"licenses", "List licenses for installed packages"},
		{"link", "Symlink a package"},
		{"list", "List installed packages"},
		{"login", "Store registry credentials"},
		{"logout", "Clear registry credentials"},
		{"outdated", "Check for outdated packages"},
		{"owner", "Manage package owners"},
		{"pack", "Create a compressed gzip archive"},
		{"plugin", "Manage plugins"},
		{"policies", "Define project-wide policies"},
		{"publish", "Publish a package to npm"},
		{"rebuild", "Rebuild all packages"},
		{"remove", "Remove a package"},
		{"run", "Run a defined package script"},
		{"set", "Change configuration settings"},
		{"start", "Run the start script"},
		{"tag", "Add, remove, or list package tags"},
		{"team", "Manage organization teams"},
		{"test", "Run the test script"},
		{"unlink", "Remove a linked package"},
		{"unplug", "Break a package out of cache"},
		{"upgrade", "Upgrade packages"},
		{"upgrade-interactive", "Interactively upgrade packages"},
		{"version", "Update the package version"},
		{"versions", "Display version information"},
		{"why", "Show information about why a package is installed"},
		{"workspace", "Run a command within a workspace"},
		{"workspaces", "Manage workspaces"},
	})

	// pnpm - Fast Node.js package manager
	s.registerWithDesc("pnpm", []subCmd{
		{"add", "Install a package"},
		{"audit", "Check for known security issues"},
		{"bin", "Print the directory where executables are installed"},
		{"config", "Manage the pnpm configuration files"},
		{"create", "Create a project from a create-* starter kit"},
		{"dedupe", "Perform deduplication"},
		{"dlx", "Run a package in a temporary environment"},
		{"env", "Manage Node.js versions"},
		{"exec", "Execute a shell command"},
		{"fetch", "Fetch packages from a lockfile"},
		{"import", "Generate pnpm-lock.yaml from package-lock.json"},
		{"init", "Create a package.json file"},
		{"install", "Install all dependencies"},
		{"licenses", "List licenses of packages"},
		{"link", "Connect local project to another"},
		{"list", "List installed packages"},
		{"outdated", "Check for outdated packages"},
		{"pack", "Create a tarball from a package"},
		{"patch", "Prepare a package for patching"},
		{"patch-commit", "Commit a patch"},
		{"prune", "Remove extraneous packages"},
		{"publish", "Publish a package to npm"},
		{"rebuild", "Rebuild a package"},
		{"remove", "Remove packages from node_modules"},
		{"root", "Print the root directory"},
		{"run", "Run a defined package script"},
		{"server", "Start a store server"},
		{"setup", "Set up pnpm"},
		{"start", "Run the start script"},
		{"store", "Manage the package store"},
		{"test", "Run the test script"},
		{"unlink", "Unlink a linked package"},
		{"update", "Update packages"},
		{"why", "Show why a package is installed"},
	})

	// Go programming language
	s.registerWithDesc("go", []subCmd{
		{"build", "Compile packages and dependencies"},
		{"bug", "Start a bug report"},
		{"clean", "Remove object files and cached files"},
		{"doc", "Show documentation for package or symbol"},
		{"env", "Print Go environment information"},
		{"fix", "Update packages to use new APIs"},
		{"fmt", "Gofmt (reformat) package sources"},
		{"generate", "Generate Go files by processing source"},
		{"get", "Add dependencies to current module"},
		{"help", "Display help for a command"},
		{"install", "Compile and install packages"},
		{"list", "List packages or modules"},
		{"mod", "Module maintenance"},
		{"run", "Compile and run Go program"},
		{"test", "Test packages"},
		{"tool", "Run specified go tool"},
		{"version", "Print Go version"},
		{"vet", "Report likely mistakes in packages"},
		{"work", "Workspace maintenance"},
	})

	// Cargo - Rust package manager
	s.registerWithDesc("cargo", []subCmd{
		{"add", "Add dependencies to a Cargo.toml manifest file"},
		{"bench", "Execute benchmarks of a package"},
		{"build", "Compile the current package"},
		{"check", "Check a local package for errors"},
		{"clean", "Remove artifacts that cargo has generated"},
		{"clippy", "Run Clippy lints"},
		{"doc", "Build package documentation"},
		{"fetch", "Fetch dependencies from network"},
		{"fix", "Automatically fix lint warnings"},
		{"fmt", "Format Rust source code"},
		{"generate-lockfile", "Generate Cargo.lock for a project"},
		{"help", "Display help for a command"},
		{"init", "Create a new Cargo package in an existing directory"},
		{"install", "Install a Rust binary"},
		{"locate-project", "Print a JSON representation of a Cargo.toml file's location"},
		{"login", "Log in to a registry"},
		{"logout", "Log out from a registry"},
		{"metadata", "Output the resolved dependencies in JSON"},
		{"new", "Create a new Cargo package"},
		{"owner", "Manage the owners of a crate"},
		{"package", "Assemble the local package into a tarball"},
		{"pkgid", "Print a fully qualified package spec"},
		{"publish", "Upload a package to the registry"},
		{"read-manifest", "Print a JSON representation of a Cargo.toml manifest"},
		{"remove", "Remove dependencies from a Cargo.toml manifest file"},
		{"report", "Generate and display various reports"},
		{"run", "Run a binary or example"},
		{"rustc", "Compile a package with extra rustc flags"},
		{"rustdoc", "Build package's documentation with rustdoc flags"},
		{"search", "Search packages in the registry"},
		{"test", "Run tests for a package"},
		{"tree", "Display a tree of dependencies"},
		{"uninstall", "Remove a Rust binary"},
		{"update", "Update dependencies in Cargo.lock"},
		{"vendor", "Vendor all dependencies locally"},
		{"verify-project", "Check correctness of crate manifest"},
		{"version", "Show version information"},
		{"yank", "Remove a pushed crate from the index"},
	})

	// pip - Python package manager
	s.registerWithDesc("pip", []subCmd{
		{"cache", "Inspect and manage pip's wheel cache"},
		{"check", "Verify installed packages have compatible dependencies"},
		{"completion", "Generate shell completion"},
		{"config", "Manage pip configuration"},
		{"debug", "Show information for debugging"},
		{"download", "Download packages"},
		{"freeze", "Output installed packages in requirements format"},
		{"hash", "Compute hashes of package archives"},
		{"help", "Show help for commands"},
		{"index", "Inspect available versions of a package"},
		{"inspect", "Inspect the environment"},
		{"install", "Install packages"},
		{"list", "List installed packages"},
		{"search", "Search PyPI for packages"},
		{"show", "Show information about installed packages"},
		{"uninstall", "Uninstall packages"},
		{"wheel", "Build wheels from your requirements"},
	})

	// Python
	s.registerWithDesc("python", []subCmd{
		{"-c", "Execute Python code from command line"},
		{"-m", "Run library module as a script"},
		{"-i", "Run in interactive mode after executing script"},
		{"-u", "Unbuffered binary stdout and stderr"},
		{"-v", "Verbose mode"},
		{"-V", "Print Python version"},
		{"--version", "Print Python version"},
		{"--help", "Print help message"},
	})

	// GitHub CLI
	s.registerWithDesc("gh", []subCmd{
		{"alias", "Create command shortcuts"},
		{"api", "Make an authenticated GitHub API request"},
		{"attestation", "Work with artifact attestations"},
		{"auth", "Authenticate gh with GitHub"},
		{"browse", "Open the repo in the browser"},
		{"cache", "Manage GitHub Actions caches"},
		{"codespace", "Connect to and manage codespaces"},
		{"completion", "Generate shell completion scripts"},
		{"config", "Manage configuration for gh"},
		{"extension", "Manage gh extensions"},
		{"gist", "Manage gists"},
		{"gpg-key", "Manage GPG keys"},
		{"help", "Display help for a command"},
		{"issue", "Manage issues"},
		{"label", "Manage labels"},
		{"org", "Manage organizations"},
		{"pr", "Manage pull requests"},
		{"project", "Work with GitHub Projects"},
		{"release", "Manage GitHub releases"},
		{"repo", "Manage repositories"},
		{"ruleset", "View repository rulesets"},
		{"run", "View and manage GitHub Actions workflow runs"},
		{"search", "Search for repositories, issues, and pull requests"},
		{"secret", "Manage GitHub secrets"},
		{"ssh-key", "Manage SSH keys"},
		{"status", "Print information about relevant issues, PRs, notifications"},
		{"variable", "Manage GitHub Actions variables"},
		{"workflow", "View and manage GitHub Actions workflows"},
	})

	// systemctl - Linux service management
	s.registerWithDesc("systemctl", []subCmd{
		{"cat", "Show unit files"},
		{"daemon-reexec", "Reexecute the manager"},
		{"daemon-reload", "Reload unit files"},
		{"disable", "Disable one or more units"},
		{"edit", "Edit unit files"},
		{"enable", "Enable one or more units"},
		{"halt", "Shut down and halt the system"},
		{"hibernate", "Hibernate the system"},
		{"hybrid-sleep", "Hybrid sleep the system"},
		{"is-active", "Check whether units are active"},
		{"is-enabled", "Check whether units are enabled"},
		{"is-failed", "Check whether units failed"},
		{"isolate", "Start one unit and stop all others"},
		{"kexec", "Shut down and reboot with kexec"},
		{"kill", "Send signal to processes of a unit"},
		{"link", "Link unit files"},
		{"list-dependencies", "Show unit dependency tree"},
		{"list-jobs", "List jobs"},
		{"list-machines", "List registered machines"},
		{"list-sockets", "List socket units"},
		{"list-timers", "List timer units"},
		{"list-unit-files", "List installed unit files"},
		{"list-units", "List loaded units"},
		{"log-level", "Get/set logging threshold"},
		{"mask", "Mask one or more units"},
		{"poweroff", "Shut down and power off the system"},
		{"preset", "Reset enabling of units to defaults"},
		{"reboot", "Shut down and reboot the system"},
		{"reenable", "Reenable one or more units"},
		{"reload", "Reload one or more units"},
		{"reload-or-restart", "Reload or restart one or more units"},
		{"rescue", "Enter rescue mode"},
		{"reset-failed", "Reset failed state"},
		{"restart", "Restart one or more units"},
		{"revert", "Revert unit file changes"},
		{"set-default", "Set the default target"},
		{"set-environment", "Set environment variables"},
		{"set-property", "Set unit properties"},
		{"show", "Show unit properties"},
		{"show-environment", "Dump environment"},
		{"start", "Start one or more units"},
		{"status", "Show runtime status of units"},
		{"stop", "Stop one or more units"},
		{"suspend", "Suspend the system"},
		{"suspend-then-hibernate", "Suspend then hibernate"},
		{"switch-root", "Switch root filesystem"},
		{"try-reload-or-restart", "Try to reload-or-restart units"},
		{"try-restart", "Try to restart units"},
		{"unmask", "Unmask one or more units"},
		{"unset-environment", "Unset environment variables"},
	})

	// apt - Debian/Ubuntu package manager
	s.registerWithDesc("apt", []subCmd{
		{"autoremove", "Remove automatically installed packages no longer needed"},
		{"build-dep", "Configure build-dependencies for source packages"},
		{"changelog", "View package changelogs"},
		{"check", "Verify that there are no broken dependencies"},
		{"clean", "Clear out the local repository of retrieved package files"},
		{"depends", "Show package dependencies"},
		{"dist-upgrade", "Upgrade the system by removing/installing packages"},
		{"download", "Download package files"},
		{"edit-sources", "Edit the source list"},
		{"full-upgrade", "Perform an upgrade, possibly installing and removing packages"},
		{"help", "Display help information"},
		{"install", "Install packages"},
		{"list", "List packages based on package names"},
		{"moo", "Easter egg"},
		{"policy", "Show policy settings"},
		{"purge", "Remove packages and configuration files"},
		{"rdepends", "Show reverse dependencies"},
		{"reinstall", "Reinstall packages"},
		{"remove", "Remove packages"},
		{"satisfy", "Satisfy dependency strings"},
		{"search", "Search package descriptions"},
		{"show", "Show package details"},
		{"showsrc", "Show source package details"},
		{"source", "Download source archives"},
		{"update", "Update list of available packages"},
		{"upgrade", "Install available upgrades for all packages"},
	})

	// Homebrew - macOS package manager
	s.registerWithDesc("brew", []subCmd{
		{"analytics", "Manage analytics preferences"},
		{"autoremove", "Uninstall unused dependencies"},
		{"cask", "Manage macOS apps"},
		{"cleanup", "Remove stale lock files and outdated downloads"},
		{"commands", "Show commands"},
		{"completions", "Manage shell completions"},
		{"config", "Show Homebrew configuration"},
		{"deps", "Show dependencies for formulae"},
		{"desc", "Show formula descriptions"},
		{"doctor", "Check for potential problems"},
		{"edit", "Edit formula or cask"},
		{"fetch", "Download source packages"},
		{"gist-logs", "Upload logs to GitHub Gist"},
		{"help", "Display help"},
		{"home", "Open formula homepage"},
		{"info", "Show formula information"},
		{"install", "Install formula or cask"},
		{"leaves", "Show installed formulae not required by another formula"},
		{"link", "Symlink a formula's files"},
		{"list", "List installed formulae"},
		{"log", "Show the git log for formula"},
		{"migrate", "Migrate formula to a different tap"},
		{"missing", "Check for missing dependencies"},
		{"options", "Show formula install options"},
		{"outdated", "Show outdated formulae or casks"},
		{"pin", "Pin a formula to a version"},
		{"postinstall", "Rerun the post-install step"},
		{"reinstall", "Reinstall formula or cask"},
		{"search", "Search for formulae and casks"},
		{"services", "Manage background services with macOS launchctl"},
		{"shellenv", "Print export statements for shell environment"},
		{"tap", "Tap a formula repository"},
		{"tap-info", "Show tap info"},
		{"uninstall", "Uninstall formula or cask"},
		{"unlink", "Remove symlinks"},
		{"unpin", "Unpin a formula"},
		{"untap", "Remove a tap"},
		{"update", "Fetch newest version of Homebrew"},
		{"upgrade", "Upgrade outdated formulae and casks"},
		{"uses", "Show formulae that depend on formula"},
	})

	// Terraform - Infrastructure as Code
	s.registerWithDesc("terraform", []subCmd{
		{"apply", "Create or update infrastructure"},
		{"console", "Interactive console for Terraform expressions"},
		{"destroy", "Destroy Terraform-managed infrastructure"},
		{"fmt", "Reformat configuration files"},
		{"force-unlock", "Release a stuck lock on the current workspace"},
		{"get", "Download and update modules"},
		{"graph", "Generate a Graphviz graph"},
		{"import", "Import existing infrastructure into Terraform"},
		{"init", "Initialize a Terraform working directory"},
		{"login", "Obtain and save credentials for a remote host"},
		{"logout", "Remove locally-stored credentials"},
		{"metadata", "Metadata related commands"},
		{"output", "Show output values from state"},
		{"plan", "Show an execution plan"},
		{"providers", "Show providers required for configuration"},
		{"refresh", "Update state to match remote systems"},
		{"show", "Show current state or a saved plan"},
		{"state", "Advanced state management"},
		{"taint", "Mark a resource instance as not fully functional"},
		{"test", "Execute integration tests"},
		{"untaint", "Remove the taint status"},
		{"validate", "Validate configuration files"},
		{"version", "Show the current Terraform version"},
		{"workspace", "Workspace management"},
	})

	// AWS CLI
	s.registerWithDesc("aws", []subCmd{
		{"acm", "AWS Certificate Manager"},
		{"apigateway", "Amazon API Gateway"},
		{"autoscaling", "Auto Scaling"},
		{"cloudformation", "AWS CloudFormation"},
		{"cloudfront", "Amazon CloudFront"},
		{"cloudwatch", "Amazon CloudWatch"},
		{"configure", "Configure AWS CLI"},
		{"dynamodb", "Amazon DynamoDB"},
		{"ec2", "Amazon Elastic Compute Cloud"},
		{"ecr", "Amazon Elastic Container Registry"},
		{"ecs", "Amazon Elastic Container Service"},
		{"eks", "Amazon Elastic Kubernetes Service"},
		{"elasticache", "Amazon ElastiCache"},
		{"elasticbeanstalk", "AWS Elastic Beanstalk"},
		{"elb", "Elastic Load Balancing"},
		{"elbv2", "Elastic Load Balancing v2"},
		{"events", "Amazon EventBridge"},
		{"help", "Display help"},
		{"iam", "AWS Identity and Access Management"},
		{"kinesis", "Amazon Kinesis"},
		{"kms", "AWS Key Management Service"},
		{"lambda", "AWS Lambda"},
		{"logs", "Amazon CloudWatch Logs"},
		{"rds", "Amazon Relational Database Service"},
		{"route53", "Amazon Route 53"},
		{"s3", "Amazon Simple Storage Service"},
		{"s3api", "Amazon S3 API"},
		{"secretsmanager", "AWS Secrets Manager"},
		{"ses", "Amazon Simple Email Service"},
		{"sns", "Amazon Simple Notification Service"},
		{"sqs", "Amazon Simple Queue Service"},
		{"ssm", "AWS Systems Manager"},
		{"sts", "AWS Security Token Service"},
		{"version", "Show CLI version"},
	})

	// Google Cloud CLI
	s.registerWithDesc("gcloud", []subCmd{
		{"access-context-manager", "Manage access policies"},
		{"ai", "Cloud AI Platform"},
		{"alpha", "Alpha commands"},
		{"anthos", "Anthos commands"},
		{"app", "App Engine management"},
		{"artifacts", "Artifact Registry"},
		{"auth", "Authentication commands"},
		{"beta", "Beta commands"},
		{"builds", "Cloud Build operations"},
		{"components", "Install and update CLI components"},
		{"composer", "Cloud Composer environments"},
		{"compute", "Compute Engine resources"},
		{"config", "CLI configuration"},
		{"container", "Container clusters and images"},
		{"dataflow", "Dataflow pipelines"},
		{"dataproc", "Dataproc clusters"},
		{"deployment-manager", "Deployment Manager"},
		{"dns", "Cloud DNS"},
		{"domains", "Cloud Domains"},
		{"endpoints", "Cloud Endpoints"},
		{"filestore", "Cloud Filestore"},
		{"firebase", "Firebase projects"},
		{"functions", "Cloud Functions"},
		{"help", "Search for help"},
		{"iam", "IAM policies and service accounts"},
		{"info", "Show CLI info"},
		{"init", "Initialize CLI"},
		{"kms", "Cloud KMS"},
		{"logging", "Cloud Logging"},
		{"memcache", "Cloud Memorystore"},
		{"ml", "AI Platform"},
		{"organizations", "Organizations"},
		{"projects", "Project management"},
		{"pubsub", "Cloud Pub/Sub"},
		{"redis", "Cloud Memorystore for Redis"},
		{"run", "Cloud Run"},
		{"scheduler", "Cloud Scheduler"},
		{"secrets", "Secret Manager"},
		{"services", "Service management"},
		{"source", "Cloud Source Repositories"},
		{"spanner", "Cloud Spanner"},
		{"sql", "Cloud SQL"},
		{"storage", "Cloud Storage"},
		{"tasks", "Cloud Tasks"},
		{"topic", "Help topics"},
		{"version", "Show CLI version"},
	})

	// Azure CLI
	s.registerWithDesc("az", []subCmd{
		{"account", "Manage Azure subscriptions"},
		{"acr", "Azure Container Registry"},
		{"ad", "Azure Active Directory"},
		{"aks", "Azure Kubernetes Service"},
		{"apim", "Azure API Management"},
		{"appconfig", "App Configuration"},
		{"appservice", "App Service"},
		{"batch", "Azure Batch"},
		{"bicep", "Bicep operations"},
		{"cdn", "Azure Content Delivery Network"},
		{"cloud", "Manage registered clouds"},
		{"cognitiveservices", "Cognitive Services"},
		{"config", "CLI configuration"},
		{"configure", "Configure CLI defaults"},
		{"container", "Container Instances"},
		{"cosmosdb", "Azure Cosmos DB"},
		{"deployment", "Deployment operations"},
		{"disk", "Managed Disks"},
		{"dns", "DNS zones and records"},
		{"eventgrid", "Event Grid"},
		{"eventhubs", "Event Hubs"},
		{"extension", "Manage CLI extensions"},
		{"feature", "Feature registration"},
		{"feedback", "Send feedback"},
		{"find", "AI-powered query"},
		{"functionapp", "Function Apps"},
		{"group", "Resource groups"},
		{"hdinsight", "HDInsight clusters"},
		{"help", "Display help"},
		{"identity", "Managed identities"},
		{"image", "VM images"},
		{"iot", "IoT Hub"},
		{"keyvault", "Key Vault"},
		{"kusto", "Azure Data Explorer"},
		{"login", "Log in to Azure"},
		{"logout", "Log out"},
		{"logicapp", "Logic Apps"},
		{"managed-cassandra", "Managed Cassandra"},
		{"maps", "Azure Maps"},
		{"mariadb", "Azure Database for MariaDB"},
		{"monitor", "Azure Monitor"},
		{"mysql", "Azure Database for MySQL"},
		{"network", "Network resources"},
		{"postgres", "Azure Database for PostgreSQL"},
		{"provider", "Resource providers"},
		{"redis", "Azure Cache for Redis"},
		{"resource", "Resource management"},
		{"role", "Role-based access control"},
		{"search", "Azure Cognitive Search"},
		{"servicebus", "Service Bus"},
		{"sf", "Service Fabric"},
		{"signalr", "Azure SignalR"},
		{"sql", "Azure SQL"},
		{"staticwebapp", "Static Web Apps"},
		{"storage", "Storage accounts"},
		{"tag", "Resource tags"},
		{"term", "Marketplace terms"},
		{"version", "Show CLI version"},
		{"vm", "Virtual Machines"},
		{"vmss", "VM Scale Sets"},
		{"webapp", "Web Apps"},
	})

	// VS Code / code
	s.registerWithDesc("code", []subCmd{
		{"-a", "Add folder to last active window"},
		{"-d", "Compare two files with each other"},
		{"-g", "Open file at specific line and character"},
		{"-n", "Force new window"},
		{"-r", "Reuse existing window"},
		{"-w", "Wait for files to be closed"},
		{"--add", "Add folder to last active window"},
		{"--diff", "Compare two files with each other"},
		{"--disable-extensions", "Disable all installed extensions"},
		{"--extensions-dir", "Set the root path for extensions"},
		{"--goto", "Open file at specific line and character"},
		{"--help", "Print usage"},
		{"--install-extension", "Install an extension"},
		{"--list-extensions", "List installed extensions"},
		{"--locale", "Set display language"},
		{"--log", "Set log level"},
		{"--new-window", "Force new window"},
		{"--prof-startup", "Profile startup time"},
		{"--reuse-window", "Reuse existing window"},
		{"--status", "Print process usage and diagnostics"},
		{"--telemetry", "Show telemetry status"},
		{"--uninstall-extension", "Uninstall an extension"},
		{"--user-data-dir", "Set user data directory"},
		{"--verbose", "Print verbose output"},
		{"--version", "Print version"},
		{"--wait", "Wait for files to be closed"},
	})

	// Vim / Neovim
	s.registerWithDesc("vim", []subCmd{
		{"+", "Start at end of file"},
		{"-c", "Execute command after loading first file"},
		{"-d", "Diff mode"},
		{"-e", "Ex mode"},
		{"-n", "No swap file"},
		{"-o", "Open windows horizontally"},
		{"-O", "Open windows vertically"},
		{"-p", "Open tab pages"},
		{"-R", "Read-only mode"},
		{"-r", "Recover crashed editing sessions"},
		{"-s", "Silent (batch) mode"},
		{"-S", "Source file after loading first file"},
		{"-u", "Use specified vimrc"},
		{"-v", "Vi mode"},
		{"--clean", "Start in a clean environment"},
		{"--help", "Print help message"},
		{"--noplugin", "Don't load any plugins"},
		{"--version", "Print version"},
	})

	s.registerWithDesc("nvim", []subCmd{
		{"+", "Start at end of file"},
		{"-c", "Execute command after loading first file"},
		{"-d", "Diff mode"},
		{"-e", "Ex mode"},
		{"-E", "Improved Ex mode"},
		{"-es", "Silent Ex mode"},
		{"-h", "Print help message"},
		{"-l", "Execute Lua script"},
		{"-n", "No swap file"},
		{"-o", "Open windows horizontally"},
		{"-O", "Open windows vertically"},
		{"-p", "Open tab pages"},
		{"-R", "Read-only mode"},
		{"-r", "Recover crashed editing sessions"},
		{"-S", "Source file after loading first file"},
		{"-u", "Use specified init.vim"},
		{"-v", "Print version"},
		{"--clean", "Start in a clean environment"},
		{"--headless", "Don't start the UI"},
		{"--noplugin", "Don't load any plugins"},
		{"--startuptime", "Write startup timing messages"},
		{"--version", "Print version"},
	})

	// tmux
	s.registerWithDesc("tmux", []subCmd{
		{"attach", "Attach to a session (alias: attach-session, a)"},
		{"attach-session", "Attach to a session"},
		{"bind-key", "Bind a key"},
		{"break-pane", "Break pane to new window"},
		{"capture-pane", "Capture the contents of a pane"},
		{"choose-buffer", "Put a pane into buffer selection mode"},
		{"choose-client", "Put a window into client selection mode"},
		{"choose-session", "Put a pane into session selection mode"},
		{"choose-tree", "Choose a session, window, or pane from a tree"},
		{"choose-window", "Put a pane into window selection mode"},
		{"clear-history", "Clear history for a pane"},
		{"clock-mode", "Enter clock mode"},
		{"command-prompt", "Open the command prompt"},
		{"confirm-before", "Run command after confirmation"},
		{"copy-mode", "Enter copy mode"},
		{"delete-buffer", "Delete a paste buffer"},
		{"detach", "Detach current client (alias: detach-client)"},
		{"detach-client", "Detach a client"},
		{"display-menu", "Display a menu"},
		{"display-message", "Display a message in the status line"},
		{"display-panes", "Display pane indicators"},
		{"display-popup", "Display a popup box over a pane"},
		{"find-window", "Search for a pattern in windows"},
		{"has-session", "Check if a session exists"},
		{"if-shell", "Execute command if shell command succeeds"},
		{"join-pane", "Join pane from another window"},
		{"kill-pane", "Kill a pane"},
		{"kill-server", "Kill tmux server"},
		{"kill-session", "Kill a session"},
		{"kill-window", "Kill a window"},
		{"last-pane", "Select the previously active pane"},
		{"last-window", "Select the previously active window"},
		{"link-window", "Link a window to another"},
		{"list-buffers", "List paste buffers"},
		{"list-clients", "List all clients"},
		{"list-commands", "List supported commands"},
		{"list-keys", "List key bindings"},
		{"list-panes", "List panes"},
		{"list-sessions", "List sessions"},
		{"list-windows", "List windows"},
		{"load-buffer", "Load a paste buffer from a file"},
		{"lock-client", "Lock a client"},
		{"lock-server", "Lock all clients"},
		{"lock-session", "Lock all clients attached to a session"},
		{"move-pane", "Move a pane"},
		{"move-window", "Move a window to another index"},
		{"new", "Create a new session (alias: new-session)"},
		{"new-session", "Create a new session"},
		{"new-window", "Create a new window"},
		{"next-layout", "Move to the next layout"},
		{"next-window", "Move to the next window"},
		{"paste-buffer", "Paste the most recent paste buffer"},
		{"pipe-pane", "Pipe output of a pane to a shell command"},
		{"previous-layout", "Move to the previous layout"},
		{"previous-window", "Move to the previous window"},
		{"refresh-client", "Refresh a client"},
		{"rename-session", "Rename a session"},
		{"rename-window", "Rename a window"},
		{"resize-pane", "Resize a pane"},
		{"resize-window", "Resize a window"},
		{"respawn-pane", "Respawn a pane"},
		{"respawn-window", "Respawn a window"},
		{"rotate-window", "Rotate positions of panes"},
		{"run-shell", "Execute a shell command"},
		{"save-buffer", "Save a paste buffer to a file"},
		{"select-layout", "Choose a pane layout"},
		{"select-pane", "Make a pane active"},
		{"select-window", "Select a window"},
		{"send-keys", "Send keystrokes to a pane"},
		{"send-prefix", "Send the prefix key"},
		{"server-info", "Show server information"},
		{"set-buffer", "Set the contents of a paste buffer"},
		{"set-environment", "Set an environment variable"},
		{"set-hook", "Set a hook"},
		{"set-option", "Set a session option"},
		{"set-window-option", "Set a window option"},
		{"show-buffer", "Display the contents of a paste buffer"},
		{"show-environment", "Show environment variables"},
		{"show-hooks", "Show hooks"},
		{"show-messages", "Show messages"},
		{"show-options", "Show session options"},
		{"show-window-options", "Show window options"},
		{"source-file", "Execute commands from a file"},
		{"split-window", "Split a pane into two"},
		{"start-server", "Start the tmux server"},
		{"suspend-client", "Suspend a client"},
		{"swap-pane", "Swap two panes"},
		{"swap-window", "Swap two windows"},
		{"switch-client", "Switch clients"},
		{"unbind-key", "Unbind a key"},
		{"unlink-window", "Unlink a window"},
		{"wait-for", "Wait for an event or trigger it"},
	})

	// curl
	s.registerWithDesc("curl", []subCmd{
		{"-d", "Send data in POST request"},
		{"-F", "Send form data"},
		{"-H", "Add header to request"},
		{"-I", "Fetch headers only"},
		{"-L", "Follow redirects"},
		{"-o", "Write output to file"},
		{"-O", "Write output to file with remote name"},
		{"-s", "Silent mode"},
		{"-u", "Server user and password"},
		{"-v", "Verbose mode"},
		{"-X", "Specify request method"},
		{"--compressed", "Request compressed response"},
		{"--connect-timeout", "Maximum time for connection"},
		{"--cookie", "Send cookies from file"},
		{"--cookie-jar", "Save cookies to file"},
		{"--data", "Send data in POST request"},
		{"--data-binary", "Send binary data"},
		{"--data-raw", "Send raw data"},
		{"--data-urlencode", "Send URL-encoded data"},
		{"--fail", "Fail silently on HTTP errors"},
		{"--form", "Send form data"},
		{"--header", "Add header to request"},
		{"--help", "Display help"},
		{"--http1.0", "Use HTTP 1.0"},
		{"--http1.1", "Use HTTP 1.1"},
		{"--http2", "Use HTTP 2"},
		{"--insecure", "Allow insecure SSL connections"},
		{"--location", "Follow redirects"},
		{"--max-time", "Maximum time allowed for transfer"},
		{"--output", "Write output to file"},
		{"--progress-bar", "Show progress bar"},
		{"--proxy", "Use proxy"},
		{"--request", "Specify request method"},
		{"--silent", "Silent mode"},
		{"--trace", "Write debug trace to file"},
		{"--upload-file", "Upload file"},
		{"--user", "Server user and password"},
		{"--user-agent", "Set User-Agent header"},
		{"--verbose", "Verbose mode"},
		{"--version", "Show version"},
	})

	// jq - JSON processor
	s.registerWithDesc("jq", []subCmd{
		{"-c", "Compact output"},
		{"-C", "Colorize output"},
		{"-e", "Set exit status based on output"},
		{"-f", "Read filter from file"},
		{"-M", "Monochrome output"},
		{"-n", "Use null as single input"},
		{"-r", "Output raw strings"},
		{"-R", "Read raw strings"},
		{"-s", "Slurp all inputs into an array"},
		{"-S", "Sort object keys"},
		{"--arg", "Set variable to string"},
		{"--argjson", "Set variable to JSON"},
		{"--compact-output", "Compact output"},
		{"--color-output", "Colorize output"},
		{"--exit-status", "Set exit status based on output"},
		{"--from-file", "Read filter from file"},
		{"--help", "Display help"},
		{"--indent", "Set indentation level"},
		{"--jsonargs", "Treat arguments as JSON"},
		{"--monochrome-output", "Monochrome output"},
		{"--null-input", "Use null as single input"},
		{"--raw-input", "Read raw strings"},
		{"--raw-output", "Output raw strings"},
		{"--slurp", "Slurp all inputs into an array"},
		{"--slurpfile", "Slurp file into variable"},
		{"--sort-keys", "Sort object keys"},
		{"--tab", "Use tabs for indentation"},
		{"--version", "Show version"},
	})

	// redis-cli
	s.registerWithDesc("redis-cli", []subCmd{
		{"-a", "Password to use when connecting"},
		{"-c", "Enable cluster mode"},
		{"-h", "Server hostname"},
		{"-n", "Database number"},
		{"-p", "Server port"},
		{"-r", "Repeat command N times"},
		{"-u", "Server URI"},
		{"-x", "Read last argument from stdin"},
		{"--askpass", "Force prompt for password"},
		{"--bigkeys", "Sample keys looking for big ones"},
		{"--cacert", "CA certificate file"},
		{"--cert", "Client certificate file"},
		{"--cluster", "Cluster manager command and arguments"},
		{"--csv", "Output in CSV format"},
		{"--dbnum", "Database number"},
		{"--eval", "Eval command"},
		{"--help", "Display help"},
		{"--hotkeys", "Sample keys looking for hot ones"},
		{"--intrinsic-latency", "Test intrinsic latency"},
		{"--key", "Private key file"},
		{"--latency", "Enter latency mode"},
		{"--latency-dist", "Show latency as spectrum"},
		{"--latency-history", "Show latency over time"},
		{"--ldb", "Enable Lua debugger"},
		{"--ldb-sync-mode", "Lua debugger sync mode"},
		{"--lru-test", "Simulate LRU cache"},
		{"--memkeys", "Sample keys looking for big ones"},
		{"--no-auth-warning", "Don't show warning when using password"},
		{"--no-raw", "Force formatted output"},
		{"--pipe", "Transfer raw Redis protocol from stdin"},
		{"--pipe-timeout", "Timeout for pipe mode"},
		{"--raw", "Force raw output"},
		{"--rdb", "Dump database to RDB file"},
		{"--replica", "Enter replica mode"},
		{"--scan", "List all keys using SCAN"},
		{"--slave", "Enter slave mode (deprecated)"},
		{"--stat", "Show continual stats"},
		{"--tls", "Enable TLS"},
		{"--user", "ACL username"},
		{"--verbose", "Enable verbose mode"},
		{"--version", "Show version"},
	})

	// psql - PostgreSQL client
	s.registerWithDesc("psql", []subCmd{
		{"-c", "Execute a single command"},
		{"-d", "Database name to connect to"},
		{"-f", "Execute commands from file"},
		{"-h", "Database server host"},
		{"-l", "List available databases"},
		{"-o", "Write query output to file"},
		{"-p", "Database server port"},
		{"-q", "Run quietly"},
		{"-s", "Single-step mode"},
		{"-t", "Print rows only"},
		{"-U", "Database user name"},
		{"-v", "Set psql variable"},
		{"-w", "Never prompt for password"},
		{"-W", "Force password prompt"},
		{"-x", "Turn on expanded output"},
		{"--command", "Execute a single command"},
		{"--csv", "Output in CSV format"},
		{"--dbname", "Database name to connect to"},
		{"--echo-all", "Echo all input from script"},
		{"--echo-errors", "Echo failed commands"},
		{"--echo-queries", "Echo commands sent to server"},
		{"--field-separator", "Field separator for unaligned output"},
		{"--file", "Execute commands from file"},
		{"--help", "Display help"},
		{"--host", "Database server host"},
		{"--html", "Turn on HTML output"},
		{"--list", "List available databases"},
		{"--log-file", "Write session log to file"},
		{"--no-align", "Unaligned output mode"},
		{"--no-password", "Never prompt for password"},
		{"--no-psqlrc", "Don't read psqlrc file"},
		{"--output", "Write query output to file"},
		{"--password", "Force password prompt"},
		{"--pset", "Set printing option"},
		{"--port", "Database server port"},
		{"--quiet", "Run quietly"},
		{"--record-separator", "Record separator for unaligned output"},
		{"--set", "Set psql variable"},
		{"--single-line", "End of line terminates command"},
		{"--single-step", "Single-step mode"},
		{"--single-transaction", "Execute as a single transaction"},
		{"--tuples-only", "Print rows only"},
		{"--username", "Database user name"},
		{"--variable", "Set psql variable"},
		{"--version", "Show version"},
	})

	// mysql
	s.registerWithDesc("mysql", []subCmd{
		{"-D", "Database to use"},
		{"-e", "Execute command and quit"},
		{"-h", "Connect to host"},
		{"-p", "Password"},
		{"-P", "Port number"},
		{"-S", "Socket file"},
		{"-u", "User for login"},
		{"-v", "Verbose mode"},
		{"--auto-rehash", "Enable automatic rehashing"},
		{"--batch", "Don't use history file"},
		{"--binary-as-hex", "Print binary values as hex"},
		{"--column-names", "Write column names in results"},
		{"--compress", "Compress data sent to server"},
		{"--database", "Database to use"},
		{"--debug", "Write debugging log"},
		{"--default-character-set", "Set default character set"},
		{"--delimiter", "Set statement delimiter"},
		{"--execute", "Execute command and quit"},
		{"--help", "Display help"},
		{"--host", "Connect to host"},
		{"--html", "Produce HTML output"},
		{"--line-numbers", "Write line numbers for errors"},
		{"--local-infile", "Enable LOAD DATA LOCAL INFILE"},
		{"--named-commands", "Enable named commands"},
		{"--no-auto-rehash", "Disable automatic rehashing"},
		{"--no-beep", "Turn off beep on error"},
		{"--one-database", "Ignore statements except for default database"},
		{"--pager", "Set pager for output"},
		{"--password", "Password"},
		{"--port", "Port number"},
		{"--prompt", "Set custom prompt"},
		{"--protocol", "Connection protocol"},
		{"--quick", "Don't cache result"},
		{"--raw", "Don't escape special characters"},
		{"--reconnect", "Reconnect if connection is lost"},
		{"--safe-updates", "Allow only UPDATE and DELETE with keys"},
		{"--secure-auth", "Refuse old protocol authentication"},
		{"--show-warnings", "Show warnings after each statement"},
		{"--sigint-ignore", "Ignore SIGINT signals"},
		{"--silent", "Silent mode"},
		{"--skip-column-names", "Don't write column names in results"},
		{"--socket", "Socket file"},
		{"--ssl", "Enable SSL"},
		{"--ssl-ca", "CA certificate file"},
		{"--ssl-cert", "Client certificate file"},
		{"--ssl-key", "Client private key file"},
		{"--table", "Output in table format"},
		{"--tee", "Append everything to file"},
		{"--unbuffered", "Flush buffer after each query"},
		{"--user", "User for login"},
		{"--verbose", "Verbose mode"},
		{"--version", "Show version"},
		{"--vertical", "Print query results vertically"},
		{"--wait", "Wait and retry if connection is down"},
		{"--xml", "Produce XML output"},
	})

	// mongosh / mongo
	s.registerWithDesc("mongosh", []subCmd{
		{"--authenticationDatabase", "Authentication database"},
		{"--authenticationMechanism", "Authentication mechanism"},
		{"--eval", "Evaluate JavaScript"},
		{"--file", "Execute script file"},
		{"--help", "Display help"},
		{"--host", "Server to connect to"},
		{"--nodb", "Start without connecting to a database"},
		{"--norc", "Don't run .mongoshrc.js"},
		{"--password", "Password for authentication"},
		{"--port", "Port to connect to"},
		{"--quiet", "Silence output from the shell"},
		{"--retryWrites", "Retry write operations on network errors"},
		{"--shell", "Run the shell after executing files"},
		{"--tls", "Use TLS connection"},
		{"--tlsAllowInvalidCertificates", "Bypass certificate validation"},
		{"--tlsAllowInvalidHostnames", "Bypass hostname validation"},
		{"--tlsCAFile", "TLS CA certificate file"},
		{"--tlsCertificateKeyFile", "TLS certificate and key file"},
		{"--username", "Username for authentication"},
		{"--verbose", "Increase verbosity"},
		{"--version", "Show version"},
	})

	// podman - Container management (Docker alternative)
	s.registerWithDesc("podman", []subCmd{
		{"attach", "Attach to a running container"},
		{"build", "Build an image from a Containerfile"},
		{"commit", "Create new image from container changes"},
		{"container", "Manage containers"},
		{"cp", "Copy files/folders between container and host"},
		{"create", "Create but do not start a container"},
		{"diff", "Inspect changes to container filesystem"},
		{"events", "Show podman events"},
		{"exec", "Run a process in a running container"},
		{"export", "Export container filesystem as a tarball"},
		{"generate", "Generate structured data based on containers"},
		{"healthcheck", "Manage healthchecks"},
		{"history", "Show history of a specified image"},
		{"image", "Manage images"},
		{"images", "List images in local storage"},
		{"import", "Import a tarball to create a filesystem image"},
		{"info", "Display podman system information"},
		{"init", "Initialize container(s)"},
		{"inspect", "Display configuration of a container or image"},
		{"kill", "Kill running containers"},
		{"load", "Load an image from a container archive"},
		{"login", "Login to a container registry"},
		{"logout", "Logout of a container registry"},
		{"logs", "Fetch logs of a container"},
		{"machine", "Manage a virtual machine"},
		{"manifest", "Manipulate manifest lists and image indexes"},
		{"network", "Manage networks"},
		{"pause", "Pause running containers"},
		{"play", "Play a pod or volume from structured data"},
		{"pod", "Manage pods"},
		{"port", "List port mappings for a container"},
		{"ps", "List containers"},
		{"pull", "Pull an image from a registry"},
		{"push", "Push an image to a specified destination"},
		{"rename", "Rename a container"},
		{"restart", "Restart containers"},
		{"rm", "Remove containers"},
		{"rmi", "Remove images from local storage"},
		{"run", "Run a command in a new container"},
		{"save", "Save image to an archive"},
		{"search", "Search registry for image"},
		{"secret", "Manage secrets"},
		{"start", "Start containers"},
		{"stats", "Display container resource usage statistics"},
		{"stop", "Stop running containers"},
		{"system", "Manage podman"},
		{"tag", "Add an additional name to a local image"},
		{"top", "Display running processes of a container"},
		{"unpause", "Unpause containers"},
		{"unshare", "Run a command in a modified user namespace"},
		{"version", "Display version information"},
		{"volume", "Manage volumes"},
		{"wait", "Block on containers"},
	})

	// Helm - Kubernetes package manager
	s.registerWithDesc("helm", []subCmd{
		{"completion", "Generate autocompletion scripts"},
		{"create", "Create a new chart with the given name"},
		{"dependency", "Manage chart dependencies"},
		{"diff", "Preview helm upgrade changes as a diff"},
		{"env", "Helm client environment information"},
		{"get", "Download extended information of a named release"},
		{"help", "Help about any command"},
		{"history", "Fetch release history"},
		{"install", "Install a chart"},
		{"lint", "Examine a chart for possible issues"},
		{"list", "List releases"},
		{"package", "Package a chart directory into a chart archive"},
		{"plugin", "Install, list, or uninstall Helm plugins"},
		{"pull", "Download a chart from a repository"},
		{"push", "Push a chart to remote"},
		{"registry", "Login to or logout from a registry"},
		{"repo", "Add, list, remove, update, and index chart repositories"},
		{"rollback", "Roll back a release to a previous revision"},
		{"search", "Search for a keyword in charts"},
		{"show", "Show information of a chart"},
		{"status", "Display the status of a named release"},
		{"template", "Locally render templates"},
		{"test", "Run tests for a release"},
		{"uninstall", "Uninstall a release"},
		{"upgrade", "Upgrade a release"},
		{"verify", "Verify that a chart has been signed and is valid"},
		{"version", "Print the client version information"},
	})
}

// subCmd holds a subcommand and its description for registration
type subCmd struct {
	value       string
	description string
}

// registerWithDesc registers a command with descriptions for its subcommands
func (s *StaticCompleter) registerWithDesc(command string, subcommands []subCmd) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var candidates []shellinput.CompletionCandidate
	for _, sub := range subcommands {
		candidates = append(candidates, shellinput.CompletionCandidate{
			Value:       sub.value,
			Description: sub.description,
		})
	}
	s.completions[command] = candidates
}

// RegisterUserCommand allows users to register custom command completions at runtime
func (s *StaticCompleter) RegisterUserCommand(command string, subcommands []UserCompletion) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var candidates []shellinput.CompletionCandidate
	for _, sub := range subcommands {
		candidates = append(candidates, shellinput.CompletionCandidate{
			Value:       sub.Value,
			Description: sub.Description,
		})
	}
	s.completions[command] = candidates
}

// loadUserCompletions loads user-defined completions from config files
func (s *StaticCompleter) loadUserCompletions() {
	// Check for config in standard locations
	configPaths := getUserCompletionConfigPaths()

	for _, configPath := range configPaths {
		if _, err := os.Stat(configPath); err == nil {
			if err := s.loadCompletionsFromFile(configPath); err == nil {
				break // Successfully loaded from this path
			}
		}
	}
}

// getUserCompletionConfigPaths returns the paths to check for user completion config
func getUserCompletionConfigPaths() []string {
	var paths []string

	// Check XDG_CONFIG_HOME first
	if xdgConfig := os.Getenv("XDG_CONFIG_HOME"); xdgConfig != "" {
		paths = append(paths, filepath.Join(xdgConfig, "bish", "completions.yaml"))
		paths = append(paths, filepath.Join(xdgConfig, "bish", "completions.json"))
	}

	// Then check home directory
	if home := os.Getenv("HOME"); home != "" {
		paths = append(paths, filepath.Join(home, ".config", "bish", "completions.yaml"))
		paths = append(paths, filepath.Join(home, ".config", "bish", "completions.json"))
		// Also check direct home directory location
		paths = append(paths, filepath.Join(home, ".bish_completions.yaml"))
		paths = append(paths, filepath.Join(home, ".bish_completions.json"))
	}

	return paths
}

// loadCompletionsFromFile loads completions from a YAML or JSON file
func (s *StaticCompleter) loadCompletionsFromFile(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var config UserCompletionConfig

	// Try YAML first (also handles JSON since YAML is a superset)
	if strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml") {
		if err := yaml.Unmarshal(data, &config); err != nil {
			return err
		}
	} else if strings.HasSuffix(path, ".json") {
		if err := json.Unmarshal(data, &config); err != nil {
			return err
		}
	} else {
		// Try YAML, then JSON
		if err := yaml.Unmarshal(data, &config); err != nil {
			if err := json.Unmarshal(data, &config); err != nil {
				return err
			}
		}
	}

	// Register user-defined completions
	for command, completions := range config.Commands {
		s.RegisterUserCommand(command, completions)
	}

	return nil
}

// ReloadUserCompletions reloads user-defined completions from config files
func (s *StaticCompleter) ReloadUserCompletions() {
	s.loadUserCompletions()
}

// GetCompletions returns completion suggestions for a command
func (s *StaticCompleter) GetCompletions(command string, args []string) []shellinput.CompletionCandidate {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Only provide completion for the first argument (subcommand)
	if len(args) == 0 {
		if candidates, ok := s.completions[command]; ok {
			return candidates
		}
	}
	// Filter by prefix
	if len(args) == 1 {
		prefix := args[0]
		if candidates, ok := s.completions[command]; ok {
			var filtered []shellinput.CompletionCandidate
			for _, c := range candidates {
				if len(c.Value) >= len(prefix) && strings.HasPrefix(c.Value, prefix) {
					filtered = append(filtered, c)
				}
			}
			return filtered
		}
	}
	return nil
}

// GetRegisteredCommands returns a sorted list of all commands that have static completions
func (s *StaticCompleter) GetRegisteredCommands() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	commands := make([]string, 0, len(s.completions))
	for cmd := range s.completions {
		commands = append(commands, cmd)
	}
	sort.Strings(commands)
	return commands
}

// HasCommand returns true if the command has registered completions
func (s *StaticCompleter) HasCommand(command string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, ok := s.completions[command]
	return ok
}
