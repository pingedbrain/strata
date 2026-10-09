Feature: Login
  Users authenticate with credentials

  Scenario: Valid login
    Given valid credentials
    When the login request is posted
    Then a session token is returned
