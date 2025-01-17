Feature: Create Player

  Scenario: Happy path - Create player with name and positions
    Given I want to create a player with the name "Leroy Jenkins"
    And the player plays the following positions
      | Positions |
      |        1B |
      |        2B |
      |        3B |
    When I submit a request to create the player
    Then I receive a successful response
    And the new player item exists in the database

  Scenario: Happy path - Create player with just name
    Given I want to create a player with the name "Luke Skywalker"
    When I submit a request to create the player
    Then I receive a successful response
    And the new player item exists in the database

  Scenario: Sad path - Try to create player with only positions
    Given the player plays the following positions
      | Positions |
      |        1B |
      |        2B |
      |        3B |
    When I submit a request to create the player
    Then I receive a bad request response
