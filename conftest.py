import requests
import pytest
import uuid

BASE_URL = "http://localhost:8081"


@pytest.fixture
def reg_user():


    unique_id = uuid.uuid4().hex[:8]
    register_data = {
        "email": f"testuser_{unique_id}@example.com",
        "password": "TestPassword123!",
        "name": f"Test User {unique_id}"
    }


    response = requests.post(f"{BASE_URL}/v1/users/register", json=register_data)

    assert response.status_code == 200, (
        f"Не удалось зарегистрировать пользователя. "
        f"Статус: {response.status_code}, Ответ: {response.text}"
    )

    response_data = response.json()

    access_token = response_data["accessToken"]
    user_id = response_data["user"]["id"]  # <-- Вот правильное обращение


    yield {
        "accessToken": access_token,
        "user_id": user_id
    }

    delete_url = f"{BASE_URL}/v1/users/{user_id}"
    headers = {"Authorization": f"Bearer {access_token}"}

    delete_response = requests.delete(delete_url, headers=headers)

    assert delete_response.status_code in (200, 204, 404), (
        f"Не удалось удалить пользователя {user_id}. "
        f"Статус: {delete_response.status_code}, Ответ: {delete_response.text}"
    )