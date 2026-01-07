Feature: Journal Management
  As a user
  I want to manage my journals and entries
  So that I can keep a personal diary

  Background:
    Given I am authenticated as user "user-123"
    And the database is initialized
    And S3 storage is configured
    And Git repositories are initialized

  Scenario: Create a new journal
    When I create a journal with title "My Daily Journal"
    Then I should receive a journal with title "My Daily Journal"
    And the journal should belong to user "user-123"

  Scenario: List my journals
    Given I have created a journal "journal-1"
    And I have created a journal "journal-2"
    When I list my journals
    Then I should receive 2 journals
    And the journals should include "journal-1"
    And the journals should include "journal-2"

  Scenario: Get a specific journal
    Given I have created a journal "journal-1"
    When I get journal "journal-1"
    Then I should receive journal "journal-1"

  Scenario: Update a journal
    Given I have created a journal "journal-1" with title "Old Title"
    When I update journal "journal-1" with title "New Title"
    Then journal "journal-1" should have title "New Title"

  Scenario: Delete a journal
    Given I have created a journal "journal-1"
    When I delete journal "journal-1"
    Then journal "journal-1" should not exist

  Scenario: Create a journal entry
    Given I have created a journal "journal-1"
    When I create an entry for date "2025-12-01" with content "# Today's Entry\n\nThis is my first entry."
    Then I should receive an entry for date "2025-12-01"
    And the entry should be stored in S3
    And the entry should be committed to Git

  Scenario: Get a journal entry
    Given I have created a journal "journal-1"
    And I have created an entry for date "2025-12-01" with content "Entry content"
    When I get entry for date "2025-12-01" from journal "journal-1"
    Then I should receive the entry content "Entry content"

  Scenario: Update a journal entry
    Given I have created a journal "journal-1"
    And I have created an entry for date "2025-12-01" with content "Old content"
    When I update entry for date "2025-12-01" with content "New content"
    Then the entry should have content "New content"
    And a new Git commit should be created

  Scenario: List journal entries
    Given I have created a journal "journal-1"
    And I have created an entry for date "2025-12-01"
    And I have created an entry for date "2025-12-02"
    When I list entries for journal "journal-1"
    Then I should receive 2 entries

  Scenario: Delete a journal entry
    Given I have created a journal "journal-1"
    And I have created an entry for date "2025-12-01"
    When I delete entry for date "2025-12-01" from journal "journal-1"
    Then the entry should not exist

  Scenario: List entry versions
    Given I have created a journal "journal-1"
    And I have created an entry for date "2025-12-01" with content "Version 1"
    And I have updated the entry with content "Version 2"
    When I list versions for entry "2025-12-01"
    Then I should receive 2 versions

  Scenario: Get a specific version
    Given I have created a journal "journal-1"
    And I have created an entry for date "2025-12-01" with content "Version 1"
    And I have updated the entry with content "Version 2"
    When I get version "commit-hash-1" for entry "2025-12-01"
    Then I should receive content "Version 1"
