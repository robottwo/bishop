# Next Priority

**#1 MCP-native-shell** - Implement Model Context Protocol (MCP) as a native shell capability

MCP is an open protocol that enables AI assistants to connect to external tools and data sources through standardized servers. By making Bishop MCP-native, the shell itself becomes an MCP host that can:

- Discover and connect to MCP servers (web search, filesystem, databases, APIs, etc.)
- Expose MCP tools directly to the AI agent as first-class shell capabilities
- Allow users to install/configure MCP servers as easily as installing shell plugins
- Provide a consistent interface for tool calling across different LLM providers

This makes Bishop extensible through the MCP ecosystem rather than requiring custom tool integrations for each data source.

# Bash Parity

- history expansion
- Job Control (Ctrl+Z, fg)

# Ergonomics

- keybind configuration (inputrc)
- syntax highlighting in shellinput

# AI

- stop using go-openai - use raw http requests
- [1.0] support custom instructions
- support agent modifying memory
- [set ollama context window](https://github.com/ollama/ollama/pull/6504)
- MCP support (→ **NEXT PRIORITY - see above**)
  - allow agent to search the web
  - allow agent to browse web urls
- training a small command prediction model

# Engineering

- limit total history size
