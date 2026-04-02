/* SPDX-License-Identifier: MPL-2.0
 * Copyright 2026 Tejus Pratap <tejzpr@gmail.com>
 *
 * See CONTRIBUTORS.md for full contributor list.
 */

package lang

import (
	"sort"
	"testing"
)

// ---------------------------------------------------------------------------
// SymbolKind tests
// ---------------------------------------------------------------------------

func TestSymbolKindString(t *testing.T) {
	tests := []struct {
		kind SymbolKind
		want string
	}{
		{KindFunction, "function"},
		{KindMethod, "method"},
		{KindClass, "class"},
		{KindType, "type"},
		{KindInterface, "interface"},
		{KindStruct, "struct"},
		{KindEnum, "enum"},
		{KindVariable, "variable"},
		{KindConstant, "constant"},
		{KindImport, "import"},
		{KindPackage, "package"},
		{KindModule, "module"},
		{KindTrait, "trait"},
		{KindProperty, "property"},
		{SymbolKind(99), "unknown"},
	}
	for _, tt := range tests {
		if got := tt.kind.String(); got != tt.want {
			t.Errorf("SymbolKind(%d).String() = %s, want %s", tt.kind, got, tt.want)
		}
	}
}

// ---------------------------------------------------------------------------
// Registry tests
// ---------------------------------------------------------------------------

func TestParserForFile(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"main.go", "go"},
		{"app.py", "python"},
		{"index.js", "javascript"},
		{"app.tsx", "typescript"},
		{"Main.java", "java"},
		{"main.c", "c"},
		{"main.cpp", "cpp"},
		{"main.rs", "rust"},
		{"Main.kt", "kotlin"},
		{"main.zig", "zig"},
		{"legacy.py2", "python2"},
		{"app.pyw", "python2"},
		{"style.css", "css"},
		{"app.scss", "css"},
		{"index.html", "html"},
		{"page.htm", "html"},
		{"config.xml", "xml"},
		{"icon.svg", "xml"},
		{"app.rb", "ruby"},
		{"index.php", "php"},
		{"Program.cs", "csharp"},
		{"README.md", ""},
	}
	for _, tt := range tests {
		p := ParserForFile(tt.path)
		if tt.want == "" {
			if p != nil {
				t.Errorf("ParserForFile(%s) = %s, want nil", tt.path, p.Name())
			}
		} else {
			if p == nil {
				t.Errorf("ParserForFile(%s) = nil, want %s", tt.path, tt.want)
			} else if p.Name() != tt.want {
				t.Errorf("ParserForFile(%s) = %s, want %s", tt.path, p.Name(), tt.want)
			}
		}
	}
}

func TestParserByName(t *testing.T) {
	p := ParserByName("python")
	if p == nil || p.Name() != "python" {
		t.Error("expected python parser")
	}
	p = ParserByName("nonexistent")
	if p != nil {
		t.Error("expected nil for unknown language")
	}
}

func TestRegisteredLanguages(t *testing.T) {
	langs := RegisteredLanguages()
	if len(langs) < 17 {
		t.Errorf("expected at least 17 languages, got %d: %v", len(langs), langs)
	}
	sort.Strings(langs)

	expected := []string{"c", "cpp", "csharp", "css", "go", "html", "java", "javascript", "kotlin", "php", "python", "python2", "ruby", "rust", "typescript", "xml", "zig"}
	for _, e := range expected {
		found := false
		for _, l := range langs {
			if l == e {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected language %s in registry", e)
		}
	}
}

func TestSupportedExtensions(t *testing.T) {
	exts := SupportedExtensions()
	if len(exts) < 17 {
		t.Errorf("expected at least 14 extensions, got %d", len(exts))
	}
}

func TestIsSupported(t *testing.T) {
	if !IsSupported("main.go") {
		t.Error("expected .go to be supported")
	}
	if IsSupported("README.md") {
		t.Error("expected .md to not be supported")
	}
}

func TestNormalizeExt(t *testing.T) {
	tests := []struct{ in, want string }{
		{".go", ".go"},
		{"go", ".go"},
		{".GO", ".go"},
		{"", ""},
	}
	for _, tt := range tests {
		if got := normalizeExt(tt.in); got != tt.want {
			t.Errorf("normalizeExt(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

// ---------------------------------------------------------------------------
// Go parser tests
// ---------------------------------------------------------------------------

func TestGoParser(t *testing.T) {
	src := `package main

import "fmt"

const MaxRetries = 3

var DefaultTimeout = 30

type Config struct {
	Host string
	Port int
}

type Handler interface {
	Handle() error
}

func main() {
	fmt.Println("hello")
}

func (c *Config) Validate() error {
	return nil
}
`
	p := ParserForFile("main.go")
	symbols := p.ExtractSymbols(src)

	assertHasSymbol(t, symbols, "main", KindPackage)
	assertHasSymbol(t, symbols, "MaxRetries", KindConstant)
	assertHasSymbol(t, symbols, "DefaultTimeout", KindVariable)
	assertHasSymbol(t, symbols, "Config", KindStruct)
	assertHasSymbol(t, symbols, "Handler", KindInterface)
	assertHasSymbol(t, symbols, "main", KindFunction)
	assertHasSymbol(t, symbols, "Validate", KindMethod)

	// Check method parent
	for _, s := range symbols {
		if s.Name == "Validate" && s.Kind == KindMethod {
			if s.Parent != "Config" {
				t.Errorf("expected Validate parent=Config, got %s", s.Parent)
			}
		}
	}
}

func TestGoParserIsTestFile(t *testing.T) {
	p := &GoParser{}
	if !p.IsTestFile("main_test.go") {
		t.Error("expected test file")
	}
	if p.IsTestFile("main.go") {
		t.Error("expected non-test file")
	}
}

// ---------------------------------------------------------------------------
// Python parser tests
// ---------------------------------------------------------------------------

func TestPythonParser(t *testing.T) {
	src := `MAX_RETRIES = 3

class AuthService:
    def __init__(self, db):
        self.db = db

    def login(self, user, password):
        return self.validate(user, password)

    async def logout(self, user):
        pass

def standalone_func():
    return True

async def async_helper():
    pass
`
	p := ParserForFile("auth.py")
	symbols := p.ExtractSymbols(src)

	assertHasSymbol(t, symbols, "MAX_RETRIES", KindConstant)
	assertHasSymbol(t, symbols, "AuthService", KindClass)
	assertHasSymbol(t, symbols, "__init__", KindMethod)
	assertHasSymbol(t, symbols, "login", KindMethod)
	assertHasSymbol(t, symbols, "logout", KindMethod)
	assertHasSymbol(t, symbols, "standalone_func", KindFunction)
	assertHasSymbol(t, symbols, "async_helper", KindFunction)

	for _, s := range symbols {
		if s.Name == "login" {
			if s.Parent != "AuthService" {
				t.Errorf("expected login parent=AuthService, got %s", s.Parent)
			}
		}
	}
}

func TestPythonParserIsTestFile(t *testing.T) {
	p := &PythonParser{}
	if !p.IsTestFile("test_auth.py") {
		t.Error("expected test file")
	}
	if !p.IsTestFile("auth_test.py") {
		t.Error("expected test file")
	}
	if p.IsTestFile("auth.py") {
		t.Error("expected non-test file")
	}
}

// ---------------------------------------------------------------------------
// JavaScript parser tests
// ---------------------------------------------------------------------------

func TestJavaScriptParser(t *testing.T) {
	src := `const API_URL = "https://api.example.com";

function handleRequest(req, res) {
    const body = req.body;
    return processData(body);
}

const fetchData = async (url) => {
    return fetch(url);
};

class UserController {
    constructor(db) {
        this.db = db;
    }

    getUser(id) {
        return this.db.find(id);
    }

    async updateUser(id, data) {
        return this.db.update(id, data);
    }
}

export function formatDate(date) {
    return date.toISOString();
}
`
	p := ParserForFile("app.js")
	symbols := p.ExtractSymbols(src)

	assertHasSymbol(t, symbols, "API_URL", KindConstant)
	assertHasSymbol(t, symbols, "handleRequest", KindFunction)
	assertHasSymbol(t, symbols, "fetchData", KindFunction)
	assertHasSymbol(t, symbols, "UserController", KindClass)
	assertHasSymbol(t, symbols, "getUser", KindMethod)
	assertHasSymbol(t, symbols, "updateUser", KindMethod)
	assertHasSymbol(t, symbols, "formatDate", KindFunction)
}

func TestJavaScriptParserIsTestFile(t *testing.T) {
	p := &JavaScriptParser{}
	if !p.IsTestFile("app.test.js") {
		t.Error("expected test file")
	}
	if !p.IsTestFile("app.spec.js") {
		t.Error("expected test file")
	}
	if p.IsTestFile("app.js") {
		t.Error("expected non-test file")
	}
}

// ---------------------------------------------------------------------------
// TypeScript parser tests
// ---------------------------------------------------------------------------

func TestTypeScriptParser(t *testing.T) {
	src := `export interface UserService {
    getUser(id: string): Promise<User>;
    deleteUser(id: string): void;
}

export type UserID = string;

export enum Role {
    Admin = "admin",
    User = "user",
}

export abstract class BaseController {
    constructor(protected db: Database) {}

    abstract handle(): void;

    protected log(msg: string): void {
        console.log(msg);
    }
}

export const MAX_PAGE_SIZE = 100;

export async function fetchUsers(): Promise<User[]> {
    return [];
}

export const processData = (data: any) => {
    return transform(data);
};
`
	p := ParserForFile("service.ts")
	symbols := p.ExtractSymbols(src)

	assertHasSymbol(t, symbols, "UserService", KindInterface)
	assertHasSymbol(t, symbols, "UserID", KindType)
	assertHasSymbol(t, symbols, "Role", KindEnum)
	assertHasSymbol(t, symbols, "BaseController", KindClass)
	assertHasSymbol(t, symbols, "log", KindMethod)
	assertHasSymbol(t, symbols, "MAX_PAGE_SIZE", KindConstant)
	assertHasSymbol(t, symbols, "fetchUsers", KindFunction)
	assertHasSymbol(t, symbols, "processData", KindFunction)
}

func TestTypeScriptParserIsTestFile(t *testing.T) {
	p := &TypeScriptParser{}
	if !p.IsTestFile("app.test.ts") {
		t.Error("expected test file")
	}
	if !p.IsTestFile("app.spec.tsx") {
		t.Error("expected test file")
	}
	if p.IsTestFile("app.ts") {
		t.Error("expected non-test file")
	}
}

// ---------------------------------------------------------------------------
// Java parser tests
// ---------------------------------------------------------------------------

func TestJavaParser(t *testing.T) {
	src := `package com.example.auth;

public class AuthService {
    private static final int MAX_RETRIES = 3;

    public boolean login(String user, String pass) {
        return validate(user, pass);
    }

    private boolean validate(String user, String pass) {
        return true;
    }
}

public interface Repository {
    void save(Object entity);
    Object findById(String id);
}

public enum Status {
    ACTIVE,
    INACTIVE
}
`
	p := ParserForFile("AuthService.java")
	symbols := p.ExtractSymbols(src)

	assertHasSymbol(t, symbols, "com.example.auth", KindPackage)
	assertHasSymbol(t, symbols, "AuthService", KindClass)
	assertHasSymbol(t, symbols, "login", KindMethod)
	assertHasSymbol(t, symbols, "validate", KindMethod)
	assertHasSymbol(t, symbols, "Repository", KindInterface)
	assertHasSymbol(t, symbols, "Status", KindEnum)
	assertHasSymbol(t, symbols, "MAX_RETRIES", KindConstant)
}

func TestJavaParserIsTestFile(t *testing.T) {
	p := &JavaParser{}
	if !p.IsTestFile("AuthServiceTest.java") {
		t.Error("expected test file")
	}
	if p.IsTestFile("AuthService.java") {
		t.Error("expected non-test file")
	}
}

// ---------------------------------------------------------------------------
// C parser tests
// ---------------------------------------------------------------------------

func TestCParser(t *testing.T) {
	src := `#define MAX_BUFFER 1024

typedef struct node {
    int value;
    struct node *next;
} Node;

enum color {
    RED,
    GREEN,
    BLUE
};

typedef int handle_t;

int process_data(int *data, int len) {
    for (int i = 0; i < len; i++) {
        data[i] *= 2;
    }
    return 0;
}

static void helper(void) {
    return;
}
`
	p := ParserForFile("main.c")
	symbols := p.ExtractSymbols(src)

	assertHasSymbol(t, symbols, "MAX_BUFFER", KindConstant)
	assertHasSymbol(t, symbols, "node", KindStruct)
	assertHasSymbol(t, symbols, "color", KindEnum)
	assertHasSymbol(t, symbols, "handle_t", KindType)
	assertHasSymbol(t, symbols, "process_data", KindFunction)
	assertHasSymbol(t, symbols, "helper", KindFunction)
}

func TestCParserIsTestFile(t *testing.T) {
	p := &CParser{}
	if !p.IsTestFile("test_main.c") {
		t.Error("expected test file")
	}
	if p.IsTestFile("main.c") {
		t.Error("expected non-test file")
	}
}

// ---------------------------------------------------------------------------
// C++ parser tests
// ---------------------------------------------------------------------------

func TestCppParser(t *testing.T) {
	src := `namespace myapp {

class Engine {
public:
    void start() {
        init();
    }

private:
    void init() {}
};

struct Config {
    int timeout;
    std::string host;
};

enum class Status {
    Running,
    Stopped
};

} // namespace myapp

int main(int argc, char **argv) {
    return 0;
}

void Engine::shutdown() {
    cleanup();
}
`
	p := ParserForFile("main.cpp")
	symbols := p.ExtractSymbols(src)

	assertHasSymbol(t, symbols, "myapp", KindModule)
	assertHasSymbol(t, symbols, "Engine", KindClass)
	assertHasSymbol(t, symbols, "Config", KindStruct)
	assertHasSymbol(t, symbols, "Status", KindEnum)
	assertHasSymbol(t, symbols, "main", KindFunction)
	assertHasSymbol(t, symbols, "shutdown", KindMethod)
}

func TestCppParserIsTestFile(t *testing.T) {
	p := &CppParser{}
	if !p.IsTestFile("test_engine.cpp") {
		t.Error("expected test file")
	}
	if p.IsTestFile("engine.cpp") {
		t.Error("expected non-test file")
	}
}

func TestCppParserExtensions(t *testing.T) {
	exts := []string{".cpp", ".cc", ".cxx", ".hpp", ".hxx", ".hh"}
	for _, ext := range exts {
		p := ParserForFile("file" + ext)
		if p == nil || p.Name() != "cpp" {
			t.Errorf("expected cpp parser for %s", ext)
		}
	}
}

// ---------------------------------------------------------------------------
// Rust parser tests
// ---------------------------------------------------------------------------

func TestRustParser(t *testing.T) {
	src := `mod utils;

pub const MAX_SIZE: usize = 1024;

pub trait Handler {
    fn handle(&self) -> Result<(), Error>;
}

pub struct Server {
    addr: String,
    port: u16,
}

impl Server {
    pub fn new(addr: String, port: u16) -> Self {
        Server { addr, port }
    }

    pub async fn start(&self) -> Result<(), Error> {
        Ok(())
    }
}

pub enum Status {
    Running,
    Stopped,
}

pub type Result<T> = std::result::Result<T, Error>;

fn helper() -> bool {
    true
}
`
	p := ParserForFile("main.rs")
	symbols := p.ExtractSymbols(src)

	assertHasSymbol(t, symbols, "utils", KindModule)
	assertHasSymbol(t, symbols, "MAX_SIZE", KindConstant)
	assertHasSymbol(t, symbols, "Handler", KindTrait)
	assertHasSymbol(t, symbols, "Server", KindStruct)
	assertHasSymbol(t, symbols, "new", KindMethod)
	assertHasSymbol(t, symbols, "start", KindMethod)
	assertHasSymbol(t, symbols, "Status", KindEnum)
	assertHasSymbol(t, symbols, "Result", KindType)
	assertHasSymbol(t, symbols, "helper", KindFunction)

	for _, s := range symbols {
		if s.Name == "new" && s.Kind == KindMethod {
			if s.Parent != "Server" {
				t.Errorf("expected new parent=Server, got %s", s.Parent)
			}
		}
	}
}

func TestRustParserIsTestFile(t *testing.T) {
	p := &RustParser{}
	if !p.IsTestFile("tests/integration.rs") {
		t.Error("expected test file")
	}
	if p.IsTestFile("main.rs") {
		t.Error("expected non-test file")
	}
}

// ---------------------------------------------------------------------------
// Kotlin parser tests
// ---------------------------------------------------------------------------

func TestKotlinParser(t *testing.T) {
	src := `package com.example.app

const val MAX_RETRIES = 3

interface Repository {
    fun findById(id: String): Entity?
    fun save(entity: Entity)
}

data class User(
    val name: String,
    val email: String
)

enum class Role {
    ADMIN,
    USER
}

class AuthService(private val repo: Repository) {
    fun login(user: String, pass: String): Boolean {
        return validate(user, pass)
    }

    private fun validate(user: String, pass: String): Boolean {
        return true
    }
}

fun topLevelFunc(): String {
    return "hello"
}

typealias UserList = List<User>
`
	p := ParserForFile("App.kt")
	symbols := p.ExtractSymbols(src)

	assertHasSymbol(t, symbols, "com.example.app", KindPackage)
	assertHasSymbol(t, symbols, "MAX_RETRIES", KindConstant)
	assertHasSymbol(t, symbols, "Repository", KindInterface)
	assertHasSymbol(t, symbols, "User", KindClass)
	assertHasSymbol(t, symbols, "Role", KindEnum)
	assertHasSymbol(t, symbols, "AuthService", KindClass)
	assertHasSymbol(t, symbols, "login", KindMethod)
	assertHasSymbol(t, symbols, "validate", KindMethod)
	assertHasSymbol(t, symbols, "topLevelFunc", KindFunction)
	assertHasSymbol(t, symbols, "UserList", KindType)
}

func TestKotlinParserIsTestFile(t *testing.T) {
	p := &KotlinParser{}
	if !p.IsTestFile("AuthServiceTest.kt") {
		t.Error("expected test file")
	}
	if p.IsTestFile("AuthService.kt") {
		t.Error("expected non-test file")
	}
}

// ---------------------------------------------------------------------------
// Zig parser tests
// ---------------------------------------------------------------------------

func TestZigParser(t *testing.T) {
	src := `const std = @import("std");

pub const MAX_SIZE: usize = 1024;

var global_state: i32 = 0;

pub const Config = struct {
    host: []const u8,
    port: u16,
};

pub const Status = enum {
    running,
    stopped,
};

pub const Result = union(enum) {
    ok: i32,
    err: []const u8,
};

pub fn init(allocator: std.mem.Allocator) !void {
    _ = allocator;
}

fn helper() bool {
    return true;
}

test "basic init" {
    try init(std.testing.allocator);
}
`
	p := ParserForFile("main.zig")
	symbols := p.ExtractSymbols(src)

	assertHasSymbol(t, symbols, "MAX_SIZE", KindConstant)
	assertHasSymbol(t, symbols, "global_state", KindVariable)
	assertHasSymbol(t, symbols, "Config", KindStruct)
	assertHasSymbol(t, symbols, "Status", KindEnum)
	assertHasSymbol(t, symbols, "Result", KindType)
	assertHasSymbol(t, symbols, "init", KindFunction)
	assertHasSymbol(t, symbols, "helper", KindFunction)
	assertHasSymbol(t, symbols, "basic init", KindFunction)
}

func TestZigParserIsTestFile(t *testing.T) {
	p := &ZigParser{}
	if !p.IsTestFile("test_main.zig") {
		t.Error("expected test file")
	}
	if p.IsTestFile("main.zig") {
		t.Error("expected non-test file")
	}
}

// ---------------------------------------------------------------------------
// Python2 parser tests
// ---------------------------------------------------------------------------

func TestPython2Parser(t *testing.T) {
	src := `MAX_RETRIES = 3

class OldStyleClass:
    def __init__(self, name):
        self.name = name

    def get_name(self):
        return self.name

class NewStyleClass(object):
    def __init__(self, value):
        self.value = value

    def process(self):
        print self.value
        return self.value

def standalone_func():
    return True
`
	p := ParserByName("python2")
	if p == nil {
		t.Fatal("python2 parser not found")
	}
	symbols := p.ExtractSymbols(src)

	assertHasSymbol(t, symbols, "MAX_RETRIES", KindConstant)
	assertHasSymbol(t, symbols, "OldStyleClass", KindClass)
	assertHasSymbol(t, symbols, "__init__", KindMethod)
	assertHasSymbol(t, symbols, "get_name", KindMethod)
	assertHasSymbol(t, symbols, "NewStyleClass", KindClass)
	assertHasSymbol(t, symbols, "process", KindMethod)
	assertHasSymbol(t, symbols, "standalone_func", KindFunction)
}

func TestPython2ParserIsTestFile(t *testing.T) {
	p := &Python2Parser{}
	if !p.IsTestFile("test_legacy.py2") {
		t.Error("expected test file")
	}
	if p.IsTestFile("legacy.py2") {
		t.Error("expected non-test file")
	}
}

func TestPython2ParserExtensions(t *testing.T) {
	for _, ext := range []string{".py2", ".pyw"} {
		p := ParserForFile("file" + ext)
		if p == nil || p.Name() != "python2" {
			t.Errorf("expected python2 parser for %s", ext)
		}
	}
}

// ---------------------------------------------------------------------------
// CSS parser tests
// ---------------------------------------------------------------------------

func TestCSSParser(t *testing.T) {
	src := `:root {
    --primary-color: #333;
    --font-size: 16px;
}

body {
    margin: 0;
}

.container {
    max-width: 1200px;
}

#main-content {
    padding: 20px;
}

@media (max-width: 768px) {
    .container {
        padding: 10px;
    }
}

@keyframes fadeIn {
    from { opacity: 0; }
    to { opacity: 1; }
}

@font-face {
    font-family: "MyFont";
    src: url("font.woff2");
}
`
	p := ParserForFile("style.css")
	symbols := p.ExtractSymbols(src)

	assertHasSymbol(t, symbols, "--primary-color", KindVariable)
	assertHasSymbol(t, symbols, "--font-size", KindVariable)
	assertHasSymbol(t, symbols, "body", KindClass)
	assertHasSymbol(t, symbols, ".container", KindClass)
	assertHasSymbol(t, symbols, "#main-content", KindClass)
	assertHasSymbol(t, symbols, "fadeIn", KindFunction)
	assertHasSymbol(t, symbols, "@font-face", KindType)
}

func TestCSSParserExtensions(t *testing.T) {
	for _, ext := range []string{".css", ".scss", ".less", ".sass"} {
		p := ParserForFile("file" + ext)
		if p == nil || p.Name() != "css" {
			t.Errorf("expected css parser for %s", ext)
		}
	}
}

// ---------------------------------------------------------------------------
// HTML parser tests
// ---------------------------------------------------------------------------

func TestHTMLParser(t *testing.T) {
	src := `<!DOCTYPE html>
<html>
<head>
    <meta name="viewport" content="width=device-width">
    <title>Test</title>
</head>
<body>
    <header id="top-nav">
        <nav>Links</nav>
    </header>
    <main id="content">
        <section id="intro">
            <h1>Hello</h1>
        </section>
        <my-component id="widget"></my-component>
    </main>
    <footer>Copyright</footer>
</body>
</html>
`
	p := ParserForFile("index.html")
	symbols := p.ExtractSymbols(src)

	assertHasSymbol(t, symbols, "DOCTYPE html", KindModule)
	assertHasSymbol(t, symbols, "header#top-nav", KindStruct)
	assertHasSymbol(t, symbols, "main#content", KindStruct)
	assertHasSymbol(t, symbols, "section#intro", KindStruct)
	assertHasSymbol(t, symbols, "my-component#widget", KindClass)
	assertHasSymbol(t, symbols, "meta:viewport", KindProperty)
}

func TestHTMLParserExtensions(t *testing.T) {
	for _, ext := range []string{".html", ".htm"} {
		p := ParserForFile("file" + ext)
		if p == nil || p.Name() != "html" {
			t.Errorf("expected html parser for %s", ext)
		}
	}
}

// ---------------------------------------------------------------------------
// XML parser tests
// ---------------------------------------------------------------------------

func TestXMLParser(t *testing.T) {
	src := `<?xml version="1.0" encoding="UTF-8"?>
<project xmlns="http://maven.apache.org/POM/4.0.0"
         xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance">
    <modelVersion>4.0.0</modelVersion>
    <groupId>com.example</groupId>
    <artifactId>myapp</artifactId>
    <dependencies>
        <dependency name="junit"/>
        <dependency name="mockito"/>
    </dependencies>
</project>
`
	p := ParserForFile("pom.xml")
	symbols := p.ExtractSymbols(src)

	assertHasSymbol(t, symbols, "xml-declaration", KindModule)
	assertHasSymbol(t, symbols, "xmlns", KindImport)
	assertHasSymbol(t, symbols, "xmlns:xsi", KindImport)
	assertHasSymbol(t, symbols, "project", KindClass)
	assertHasSymbol(t, symbols, "dependency[junit]", KindProperty)
	assertHasSymbol(t, symbols, "dependency[mockito]", KindProperty)
}

func TestXMLParserExtensions(t *testing.T) {
	for _, ext := range []string{".xml", ".xsl", ".xslt", ".xsd", ".svg", ".plist", ".xaml"} {
		p := ParserForFile("file" + ext)
		if p == nil || p.Name() != "xml" {
			t.Errorf("expected xml parser for %s", ext)
		}
	}
}

// ---------------------------------------------------------------------------
// Ruby parser tests
// ---------------------------------------------------------------------------

func TestRubyParser(t *testing.T) {
	src := `module Authentication
  MAX_ATTEMPTS = 3

  class AuthService
    attr_reader :db, :logger

    def initialize(db)
      @db = db
    end

    def login(user, password)
      validate(user, password)
    end

    private

    def validate(user, password)
      true
    end
  end

  def self.configure
    yield config
  end
end

class User
  attr_accessor :name, :email

  def to_s
    name
  end
end

def standalone_helper
  true
end
`
	p := ParserForFile("auth.rb")
	symbols := p.ExtractSymbols(src)

	assertHasSymbol(t, symbols, "Authentication", KindModule)
	assertHasSymbol(t, symbols, "MAX_ATTEMPTS", KindConstant)
	assertHasSymbol(t, symbols, "AuthService", KindClass)
	assertHasSymbol(t, symbols, "db", KindProperty)
	assertHasSymbol(t, symbols, "initialize", KindMethod)
	assertHasSymbol(t, symbols, "login", KindMethod)
	assertHasSymbol(t, symbols, "validate", KindMethod)
	assertHasSymbol(t, symbols, "User", KindClass)
	assertHasSymbol(t, symbols, "name", KindProperty)
	assertHasSymbol(t, symbols, "to_s", KindMethod)
	assertHasSymbol(t, symbols, "standalone_helper", KindFunction)
}

func TestRubyParserIsTestFile(t *testing.T) {
	p := &RubyParser{}
	if !p.IsTestFile("auth_test.rb") {
		t.Error("expected test file")
	}
	if !p.IsTestFile("auth_spec.rb") {
		t.Error("expected spec file")
	}
	if p.IsTestFile("auth.rb") {
		t.Error("expected non-test file")
	}
}

func TestRubyParserExtensions(t *testing.T) {
	for _, ext := range []string{".rb", ".rake", ".gemspec"} {
		p := ParserForFile("file" + ext)
		if p == nil || p.Name() != "ruby" {
			t.Errorf("expected ruby parser for %s", ext)
		}
	}
}

// ---------------------------------------------------------------------------
// PHP parser tests
// ---------------------------------------------------------------------------

func TestPHPParser(t *testing.T) {
	src := `<?php
namespace App\Services;

define('VERSION', '1.0');

interface Repository {
    public function findById(string $id): ?Entity;
    public function save(Entity $entity): void;
}

trait Loggable {
    public function log(string $msg): void {
        echo $msg;
    }
}

abstract class BaseService {
    const MAX_RETRIES = 3;

    abstract public function execute(): void;

    protected function retry(callable $fn): mixed {
        return $fn();
    }
}

class AuthService extends BaseService {
    use Loggable;

    public function execute(): void {
        $this->login();
    }

    public function login(): bool {
        return true;
    }

    private function validate(string $user): bool {
        return true;
    }
}

enum Status {
    case Active;
    case Inactive;
}

function helper(): bool {
    return true;
}
`
	p := ParserForFile("AuthService.php")
	symbols := p.ExtractSymbols(src)

	assertHasSymbol(t, symbols, `App\Services`, KindPackage)
	assertHasSymbol(t, symbols, "VERSION", KindConstant)
	assertHasSymbol(t, symbols, "Repository", KindInterface)
	assertHasSymbol(t, symbols, "Loggable", KindTrait)
	assertHasSymbol(t, symbols, "BaseService", KindClass)
	assertHasSymbol(t, symbols, "MAX_RETRIES", KindConstant)
	assertHasSymbol(t, symbols, "execute", KindMethod)
	assertHasSymbol(t, symbols, "AuthService", KindClass)
	assertHasSymbol(t, symbols, "login", KindMethod)
	assertHasSymbol(t, symbols, "validate", KindMethod)
	assertHasSymbol(t, symbols, "Status", KindEnum)
	assertHasSymbol(t, symbols, "helper", KindFunction)

	for _, s := range symbols {
		if s.Name == "login" && s.Kind == KindMethod {
			if s.Parent != "AuthService" {
				t.Errorf("expected login parent=AuthService, got %s", s.Parent)
			}
		}
	}
}

func TestPHPParserIsTestFile(t *testing.T) {
	p := &PHPParser{}
	if !p.IsTestFile("AuthServiceTest.php") {
		t.Error("expected test file")
	}
	if p.IsTestFile("AuthService.php") {
		t.Error("expected non-test file")
	}
}

func TestPHPParserExtensions(t *testing.T) {
	for _, ext := range []string{".php", ".phtml"} {
		p := ParserForFile("file" + ext)
		if p == nil || p.Name() != "php" {
			t.Errorf("expected php parser for %s", ext)
		}
	}
}

// ---------------------------------------------------------------------------
// C# parser tests
// ---------------------------------------------------------------------------

func TestCSharpParser(t *testing.T) {
	src := `using System;
using System.Collections.Generic;

namespace MyApp.Services
{
    public interface IRepository
    {
        void Save(object entity);
        object FindById(string id);
    }

    public enum Status
    {
        Active,
        Inactive
    }

    public struct Point
    {
        public int X { get; set; }
        public int Y { get; set; }
    }

    public abstract class BaseService
    {
        public const int MaxRetries = 3;

        public event EventHandler Changed;

        public abstract void Execute();

        protected virtual void OnChanged()
        {
            Changed?.Invoke(this, EventArgs.Empty);
        }
    }

    public class AuthService : BaseService
    {
        public string Name { get; set; }

        public override void Execute()
        {
            Login("admin", "pass");
        }

        public bool Login(string user, string pass)
        {
            return Validate(user, pass);
        }

        private bool Validate(string user, string pass)
        {
            return true;
        }
    }

    public delegate void ActionHandler(string action);

    public record UserRecord(string Name, string Email);
}
`
	p := ParserForFile("AuthService.cs")
	symbols := p.ExtractSymbols(src)

	assertHasSymbol(t, symbols, "System", KindImport)
	assertHasSymbol(t, symbols, "System.Collections.Generic", KindImport)
	assertHasSymbol(t, symbols, "MyApp.Services", KindPackage)
	assertHasSymbol(t, symbols, "IRepository", KindInterface)
	assertHasSymbol(t, symbols, "Status", KindEnum)
	assertHasSymbol(t, symbols, "Point", KindStruct)
	assertHasSymbol(t, symbols, "BaseService", KindClass)
	assertHasSymbol(t, symbols, "MaxRetries", KindConstant)
	assertHasSymbol(t, symbols, "AuthService", KindClass)
	assertHasSymbol(t, symbols, "Login", KindMethod)
	assertHasSymbol(t, symbols, "Validate", KindMethod)
	assertHasSymbol(t, symbols, "ActionHandler", KindType)
	assertHasSymbol(t, symbols, "UserRecord", KindClass)

	for _, s := range symbols {
		if s.Name == "Login" && s.Kind == KindMethod {
			if s.Parent != "AuthService" {
				t.Errorf("expected Login parent=AuthService, got %s", s.Parent)
			}
		}
	}
}

func TestCSharpParserIsTestFile(t *testing.T) {
	p := &CSharpParser{}
	if !p.IsTestFile("AuthServiceTest.cs") {
		t.Error("expected test file")
	}
	if p.IsTestFile("AuthService.cs") {
		t.Error("expected non-test file")
	}
}

func TestCSharpParserExtensions(t *testing.T) {
	for _, ext := range []string{".cs", ".csx"} {
		p := ParserForFile("file" + ext)
		if p == nil || p.Name() != "csharp" {
			t.Errorf("expected csharp parser for %s", ext)
		}
	}
}

// ---------------------------------------------------------------------------
// Edge cases
// ---------------------------------------------------------------------------

func TestEmptyContent(t *testing.T) {
	langs := RegisteredLanguages()
	for _, name := range langs {
		p := ParserByName(name)
		symbols := p.ExtractSymbols("")
		if len(symbols) != 0 {
			t.Errorf("%s: expected no symbols for empty content, got %d", name, len(symbols))
		}
	}
}

func TestSingleLineContent(t *testing.T) {
	p := ParserForFile("main.go")
	symbols := p.ExtractSymbols("package main")
	assertHasSymbol(t, symbols, "main", KindPackage)
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func assertHasSymbol(t *testing.T, symbols []Symbol, name string, kind SymbolKind) {
	t.Helper()
	for _, s := range symbols {
		if s.Name == name && s.Kind == kind {
			return
		}
	}
	t.Errorf("expected symbol %s (%s) not found in %d symbols", name, kind, len(symbols))
	for _, s := range symbols {
		t.Logf("  have: %s (%s) at line %d", s.Name, s.Kind, s.StartLine)
	}
}
