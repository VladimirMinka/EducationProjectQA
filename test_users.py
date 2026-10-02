from urllib import response

import requests
import pytest
import uuid

BASE_URL = "http://localhost:8081"


@pytest.fixture
def reg_user():
    unique_id = uuid.uuid4().hex[:8]
    register_data = {
        "email": f"test_user{unique_id}@test.com",
        "password": "password",
        "name": f"name{unique_id}test_name"
    }

    response = requests.post(f"{BASE_URL}/v1/users/register", json=register_data)

    assert response.status_code == 200, f"Ошибка регистрации: {response.text}"

    response_data = response.json()
    access_token = response_data["accessToken"]
    user_id = response_data["user"]["id"]

    yield {
        "user_id": user_id,
        "accessToken": access_token
    }

    delete_url = f"{BASE_URL}/v1/users/{user_id}"
    headers = {"Authorization": f"Bearer {access_token}"}
    delete_response = requests.delete(delete_url, headers=headers)

    assert delete_response.status_code in (200, 204), f"Не удалось удалить пользователя: {delete_response.text}"


def test_registered_user_can_access_profile(reg_user):
    token = reg_user["accessToken"]
    user_id = reg_user["user_id"]

    response = requests.get(
        f"{BASE_URL}/v1/users/{user_id}",
        headers={"Authorization": f"Bearer {token}"}
    )

    assert response.status_code == 200, f"Статус: {response.status_code}, Ответ: {response.text}"

    data = response.json()
    assert data["user"]["id"] == user_id, "ID пользователя в ответе не совпадает"


def test_add_item_to_cart(reg_user):
    token = reg_user["accessToken"]
    user_id = reg_user["user_id"]
    url = f"{BASE_URL}/v1/users/{user_id}/cart/items"
    headers = {"Authorization": f"Bearer {token}"}
    cart_item_data = {
        "product_id": "550e8400-e29b-41d4-a716-446655440001",
        "quantity": 10
    }
    response = requests.post(url, json=cart_item_data, headers=headers)
    assert response.status_code == 200, f"Cтатус: {response.status_code}, Ответ: {response.text}"
    data = response.json()
    assert len(data["items"]) > 0, "Корзина пуста, товар не добавился"
    assert data["items"][0]["productId"] == "550e8400-e29b-41d4-a716-446655440001"

def test_apply_promocode_with_session(reg_user):
    token = reg_user["accessToken"]
    user_id = reg_user["user_id"]
    session = requests.Session()
    session.headers.update({"Authorization": f"Bearer {token}"})
    cart_url = f"{BASE_URL}/v1/users/{user_id}/cart/items"
    cart_item_data = {
        "product_id": "550e8400-e29b-41d4-a716-446655440001",
        "quantity": 1
    }
    cart_response = session.post(cart_url, json=cart_item_data)
    assert cart_response.status_code == 200
    promocode = f"{BASE_URL}/v1/users/{user_id}/cart/promocode"
    promocode_data = {"code": "SAVE10"}
    promocode_response = session.post(promocode, json=promocode_data)
    assert promocode_response.status_code == 200
    data = promocode_response.json()
    assert data["appliedPromocode"] == "SAVE10"
    session.close()