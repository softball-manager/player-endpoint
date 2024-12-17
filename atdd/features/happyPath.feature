Feature: Happy Path

  Scenario: Call Local Endpoint
    Given I have a post request with body createRequestBody.json
    When I call the post endpoint to create a player
    Then the response should match successfulCreateResponse.json
    And the new player item exists in the database
