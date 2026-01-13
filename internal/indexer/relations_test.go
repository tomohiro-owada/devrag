package indexer

import (
	"testing"
)

func TestNewRelationExtractor(t *testing.T) {
	re, err := NewRelationExtractor()
	if err != nil {
		t.Fatalf("Failed to create relation extractor: %v", err)
	}
	defer re.Close()

	// Check parsers were created
	expectedLangs := []string{"go", "python", "typescript", "javascript", "php", "rust"}
	for _, lang := range expectedLangs {
		if _, ok := re.parsers[lang]; !ok {
			t.Errorf("Parser not found for language: %s", lang)
		}
	}
}

func TestExtractGoRelations(t *testing.T) {
	re, err := NewRelationExtractor()
	if err != nil {
		t.Fatalf("Failed to create relation extractor: %v", err)
	}
	defer re.Close()

	goCode := []byte(`
package main

import (
	"fmt"
	"strings"
)

func main() {
	fmt.Println("Hello")
	s := strings.ToUpper("test")
	helper()
}

func helper() {
	doSomething()
}
`)

	relations, err := re.ExtractRelations(goCode, "go", "main.go", "main", 8)
	if err != nil {
		t.Fatalf("Failed to extract relations: %v", err)
	}

	// Check imports
	hasImport := false
	for _, r := range relations {
		if r.RelationType == RelationTypeImports && r.TargetName == "fmt" {
			hasImport = true
		}
	}
	if !hasImport {
		t.Error("Expected to find import 'fmt'")
	}

	// Check function calls
	hasCalls := false
	for _, r := range relations {
		if r.RelationType == RelationTypeCalls && (r.TargetName == "Println" || r.TargetName == "helper" || r.TargetName == "doSomething") {
			hasCalls = true
		}
	}
	if !hasCalls {
		t.Error("Expected to find function calls")
	}
}

func TestExtractPythonRelations(t *testing.T) {
	re, err := NewRelationExtractor()
	if err != nil {
		t.Fatalf("Failed to create relation extractor: %v", err)
	}
	defer re.Close()

	pyCode := []byte(`
import os
from collections import defaultdict

class Animal:
    pass

class Dog(Animal):
    def bark(self):
        print("Woof")
        os.getcwd()

def main():
    d = Dog()
    d.bark()
`)

	relations, err := re.ExtractRelations(pyCode, "python", "main.py", "main", 13)
	if err != nil {
		t.Fatalf("Failed to extract relations: %v", err)
	}

	// Check imports
	hasImport := false
	for _, r := range relations {
		if r.RelationType == RelationTypeImports && (r.TargetName == "os" || r.TargetName == "collections") {
			hasImport = true
		}
	}
	if !hasImport {
		t.Error("Expected to find imports")
	}

	// Check inheritance
	hasInherit := false
	for _, r := range relations {
		if r.RelationType == RelationTypeInherits && r.TargetName == "Animal" {
			hasInherit = true
		}
	}
	if !hasInherit {
		t.Error("Expected to find inheritance from Animal")
	}
}

func TestExtractTypeScriptRelations(t *testing.T) {
	re, err := NewRelationExtractor()
	if err != nil {
		t.Fatalf("Failed to create relation extractor: %v", err)
	}
	defer re.Close()

	tsCode := []byte(`
import { User } from './user';
import * as utils from './utils';

interface Greeter {
    greet(): string;
}

class Person implements Greeter {
    greet(): string {
        console.log("Hello");
        return utils.format("Hi");
    }
}

class Employee extends Person {
    work() {
        this.greet();
    }
}
`)

	relations, err := re.ExtractRelations(tsCode, "typescript", "main.ts", "Employee", 15)
	if err != nil {
		t.Fatalf("Failed to extract relations: %v", err)
	}

	// Check imports
	hasImport := false
	for _, r := range relations {
		if r.RelationType == RelationTypeImports {
			hasImport = true
		}
	}
	if !hasImport {
		t.Error("Expected to find imports")
	}

	// Check function calls
	hasCalls := false
	for _, r := range relations {
		if r.RelationType == RelationTypeCalls && (r.TargetName == "log" || r.TargetName == "format" || r.TargetName == "greet") {
			hasCalls = true
		}
	}
	if !hasCalls {
		t.Error("Expected to find function calls")
	}
}

func TestExtractPHPRelations(t *testing.T) {
	re, err := NewRelationExtractor()
	if err != nil {
		t.Fatalf("Failed to create relation extractor: %v", err)
	}
	defer re.Close()

	phpCode := []byte(`<?php
namespace App\Services;

use App\Repository\UserRepository;
use App\Models\User;

class UserService
{
    private UserRepository $userRepository;

    public function findById(int $id): ?User
    {
        return $this->userRepository->find($id);
    }

    public function createUser(array $data): User
    {
        Log::info('Creating user');
        $user = new User();
        $this->userRepository->save($user);
        return $user;
    }
}
`)

	relations, err := re.ExtractRelations(phpCode, "php", "UserService.php", "UserService", 7)
	if err != nil {
		t.Fatalf("Failed to extract relations: %v", err)
	}

	// Check imports (use statements)
	hasImport := false
	for _, r := range relations {
		if r.RelationType == RelationTypeImports && (r.TargetName == "App\\Repository\\UserRepository" || r.TargetName == "App\\Models\\User") {
			hasImport = true
		}
	}
	if !hasImport {
		t.Error("Expected to find imports")
	}

	// Check function calls
	hasCalls := false
	for _, r := range relations {
		if r.RelationType == RelationTypeCalls && (r.TargetName == "find" || r.TargetName == "save" || r.TargetName == "info") {
			hasCalls = true
		}
	}
	if !hasCalls {
		t.Error("Expected to find function calls")
	}
}

func TestExtractRustRelations(t *testing.T) {
	re, err := NewRelationExtractor()
	if err != nil {
		t.Fatalf("Failed to create relation extractor: %v", err)
	}
	defer re.Close()

	rustCode := []byte(`
use std::collections::HashMap;
use std::sync::Arc;

pub struct UserService {
    repository: Box<dyn UserRepository>,
}

impl UserService {
    pub fn new(repository: Box<dyn UserRepository>) -> Self {
        UserService { repository }
    }

    pub fn get_user(&self, id: u64) -> Option<User> {
        self.repository.find(id)
    }

    pub fn create_user(&mut self, name: String) -> Result<User, Error> {
        let user = User::new(name);
        self.repository.save(user.clone())?;
        Ok(user)
    }
}
`)

	relations, err := re.ExtractRelations(rustCode, "rust", "user_service.rs", "UserService", 5)
	if err != nil {
		t.Fatalf("Failed to extract relations: %v", err)
	}

	// Check imports (use statements)
	hasImport := false
	for _, r := range relations {
		if r.RelationType == RelationTypeImports && (r.TargetName == "std::collections::HashMap" || r.TargetName == "std::sync::Arc") {
			hasImport = true
		}
	}
	if !hasImport {
		t.Error("Expected to find imports")
	}

	// Check function calls
	hasCalls := false
	for _, r := range relations {
		if r.RelationType == RelationTypeCalls && (r.TargetName == "find" || r.TargetName == "save" || r.TargetName == "new" || r.TargetName == "clone") {
			hasCalls = true
		}
	}
	if !hasCalls {
		t.Error("Expected to find function calls")
	}
}

func TestCleanRelationName(t *testing.T) {
	tests := []struct {
		input    string
		relType  RelationType
		expected string
	}{
		{`"fmt"`, RelationTypeImports, "fmt"},
		{`'path/to/module'`, RelationTypeImports, "path/to/module"},
		{"len", RelationTypeCalls, ""},     // Built-in, should be filtered
		{"make", RelationTypeCalls, ""},    // Built-in, should be filtered
		{"MyFunc", RelationTypeCalls, "MyFunc"},
		{"", RelationTypeCalls, ""},
	}

	for _, tc := range tests {
		result := cleanRelationName(tc.input, tc.relType)
		if result != tc.expected {
			t.Errorf("cleanRelationName(%q, %s): expected %q, got %q", tc.input, tc.relType, tc.expected, result)
		}
	}
}
