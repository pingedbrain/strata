from pytest_bdd import given, scenarios, then, when

scenarios("../features/login.feature")


@given("valid credentials")
def valid_creds():
    return {"user": "dev", "pass": "s3cret"}


@when("the login request is posted")
def post_login():
    pass


@then("a session token is returned")
def check_token():
    pass
