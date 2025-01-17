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
