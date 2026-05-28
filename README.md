# ozon_task

Тестовое задание на стажировку в Ozon

###### Описание
- Даталогическая модель БД Postgres находится в `docs/db-model.dbml` в формате DBML.
- GraphQL схема находится в `docs/schema.graphql`
- Реализовано два варианта хранилища, in-memory и postgres.
- Реализована миграция данных для PostgreSQL
- Основные таргеты для тестирования:
    - `make docker-up` - запуск варианта с PostgreSQL через docker-conpose.yml
    - `make run-memory` - запуск варианта с in-memory вариантом
    - `make test` - прогон юнит-тестов
    - `make dump-tables` - дамп таблиц PostgreSQL в которых хранятся посты и комментарии
- GraphQL UI - `http://localhost:8080`



###### Проверка с внешней БД Postgres

Шаг 1 - Поднимаем контейнеры
```
make docker-up
```

Шаг 2 - Заходим на `http://localhost:8080` в браузере

Шаг 3 - Создаем тестовый пост

```
mutation {
  createPost(input: {
    authorName: "sanek"
    title: "demo post"
    body: "some text"
    commentsEnabled: true
  }) {
    id
    title
    commentsEnabled
    createdAt
  }
}
```

Запоминаем id в ответе

Шаг 4 - Создаем корневой комментарий:

```
mutation {
  createComment(input: {
    postId: "1"
    authorName: "alex"
    body: "root comment"
  }) {
    id
    postId
    parentId
    body
  }
}
```

Запоминаем id комментария

Шаг 5 - Создаем вложенный комментарий для комментария из шага 4

```
mutation {
  createComment(input: {
    postId: "1"
    parentId: "2"
    authorName: "roman"
    body: "hater reply comment"
  }) {
    id
    postId
    parentId
    body
  }
}
```

Шаг 6 - Читаем пост с комментариями

```
query {
  post(id: "1") {
    id
    title
    commentsEnabled
    comments(first: 10) {
      edges {
        cursor
        node {
          id
          body
          replies(first: 10) {
            edges {
              node {
                id
                parentId
                body
              }
            }
          }
        }
      }
      pageInfo {
        hasNextPage
        endCursor
      }
    }
  }
}
```