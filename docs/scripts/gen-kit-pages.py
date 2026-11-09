"""Generate docs/gorock-kit/<module>.md from gorock-kit's README_RU.md files.

The module READMEs in gorock-kit are the source of truth for the API. This
script copies them into the site and adds a "Место в GOROCK" block on top, so
the two never drift apart. Re-run it after changing a module README:

    python docs/scripts/gen-kit-pages.py ../gorock-kit

The first argument is the path to a gorock-kit checkout (default ../gorock-kit
relative to the gorock repo root).
"""
import os
import re
import sys

ROOT = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
KIT = os.path.abspath(sys.argv[1] if len(sys.argv) > 1 else os.path.join(ROOT, '..', 'gorock-kit'))
OUT = os.path.join(ROOT, 'docs', 'gorock-kit')
REPO = 'https://github.com/arzab/gorock-kit'

# module -> (layer badge, where it lives in a GOROCK project)
PLACES = {
    'rockengine': (
        'Engine',
        'Сердце `cmd/main`: запускает и останавливает всё приложение. `services.Register` '
        'регистрирует сервисы как **ресурсы**, `apps.Register` — приложения как **App**. '
        'Контракты `App` и `Resource` — это и есть интерфейсы слоёв Engine и Services. '
        '→ [Main](/architecture/main), [Apps](/architecture/apps)',
    ),
    'rockconfig': (
        'Configs',
        'Загружает единый файл конфига в агрегированный `Configs` в `cmd/main`. '
        'Каждый слой объявляет свой `Configs`, rockconfig проверяет их все разом. '
        '→ [Конфигурация](/architecture/configs)',
    ),
    'rocklog': (
        'Infra',
        'Глобальный логгер проекта. `pkg/infra/logs` переводит настройки из файла в `rocklog.Init`, '
        'дальше все слои пишут через `rocklog`. → [Infra](/architecture/infra)',
    ),
    'rockfiber': (
        'Engine · Apps',
        'Готовый HTTP App: регистрируется в `cmd/apps`, принимает endpoints из `internal/delivery/endpoints`. '
        'В delivery используются его хелперы: `GET/POST`, `DefaultHandler`, `GetFromContext`, `NewError`. '
        '→ [Apps](/architecture/apps), [Delivery](/architecture/delivery)',
    ),
    'rockbus': (
        'Engine · Apps',
        'Шина событий внутри процесса — App в `cmd/apps`. Подписки описываются в '
        '`internal/delivery/events`, публикация — из любого delivery через `rockbus.PublishAsync`. '
        '→ [Apps](/architecture/apps)',
    ),
    'rockcron': (
        'Engine · Apps',
        'Планировщик — App в `cmd/apps`. Задачи описываются в `internal/delivery/jobs`. '
        '→ [Apps](/architecture/apps)',
    ),
    'rocktelebot': (
        'Engine · Apps',
        'Telegram-бот — App в `cmd/apps`. Команды и обработчики описываются в `internal/delivery`. '
        '→ [Apps](/architecture/apps)',
    ),
    'rockbun': (
        'Toolkit · Services',
        'Готовый сервис PostgreSQL. Обычно — дочерний сервис внутри `pkg/services/repository`: '
        'repository описывает функционал проекта, rockbun даёт пул и запросы. Реализует `rockengine.Resource`. '
        '→ [Services](/architecture/services)',
    ),
    'rockredis': (
        'Toolkit · Services',
        'Готовый сервис Redis для `pkg/services` (кэш, блокировки, pub/sub). Реализует `rockengine.Resource`. '
        '→ [Services](/architecture/services)',
    ),
}


def page(module: str) -> str:
    src = os.path.join(KIT, module, 'README_RU.md')
    with open(src, encoding='utf-8') as f:
        text = f.read()

    lines = text.splitlines()
    assert lines[0].startswith('# '), f'{src}: first line must be the H1'
    title, body = lines[0], '\n'.join(lines[1:])

    # Links between modules point at sibling pages on the site.
    body = re.sub(r'\]\(\.\./(rock[a-z]+)/?\)', r'](./\1)', body)

    layer, place = PLACES[module]
    header = (
        f'{title}\n\n'
        f'<!-- Сгенерировано docs/scripts/gen-kit-pages.py из {module}/README_RU.md — правьте README в gorock-kit. -->\n\n'
        f'::: tip Место в GOROCK · {layer}\n{place}\n:::\n\n'
        f'```sh\ngo get github.com/arzab/gorock-kit/{module}@latest\n```\n\n'
        f'[Исходный код и README на GitHub]({REPO}/tree/main/{module})\n'
    )
    return header + body.rstrip() + '\n'


def main() -> None:
    os.makedirs(OUT, exist_ok=True)
    for module in PLACES:
        with open(os.path.join(OUT, f'{module}.md'), 'w', encoding='utf-8', newline='\n') as f:
            f.write(page(module))
        print(f'wrote gorock-kit/{module}.md')


if __name__ == '__main__':
    main()
