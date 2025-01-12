Feature: Happy Path

  Scenario: Call Local Endpoint
    Given I want to create a player with the name "Leroy"
    And the player plays the following positions
      | Positions |
      |        1B |
      |        2B |
      |        3B |
    When I submit a request to create the player
    Then the new player item exists in the database
