Feature: Get Player

  Scenario: Happy path - Get player
    Given there is a player with the name "Leroy" who plays
      | Positions |
      | CF        |
      | RF        |
      | C         |
    When I submit a request to get the player
    Then I receive a successful response
    And the get response body is validated

  Scenario: Happy path - Get player with no positions
    Given there is a player with the name "Leroy" who has no defined positions
    When I submit a request to get the player
    Then I receive a successful response
    And the get response body is validated

  Scenario: Sad path - Player doesn't exist
    Given there is a player that doesn't exist
    When I submit a request to get the player
    Then I receive a not found response
