"""Generate the module pages of the site from the module READMEs.

The README_RU.md files in gorock-kit (the core) and gorock-catalog are the
source of truth for the API. This script copies them into the site and adds a
"Место в GOROCK" block on top, so the two never drift apart. Re-run it after
changing a module README:

    python docs/scripts/gen-kit-pages.py ../gorock-kit ../gorock-catalog

The arguments are paths to the checkouts (default: siblings of the gorock
repo).
"""
import os
import re
import sys

ROOT = os.path.dirname(os.path.dirname(os.path.dirname(os.path.abspath(__file__))))
KIT = os.path.abspath(sys.argv[1] if len(sys.argv) > 1 else os.path.join(ROOT, '..', 'gorock-kit'))
CATALOG = os.path.abspath(sys.argv[2] if len(sys.argv) > 2 else os.path.join(ROOT, '..', 'gorock-catalog'))

# repo -> (checkout, site section, GitHub repo)
REPOS = {
    'kit': (KIT, 'gorock-kit', 'https://github.com/arzab/gorock-kit'),
    'catalog': (CATALOG, 'catalog', 'https://github.com/arzab/gorock-catalog'),
}

LEGACY = ('::: warning Перенесён из gorock-kit v0.2.0 как есть\n'
          'Код и API — те же, что были в `github.com/arzab/gorock-kit/{m}@v0.2.0`, путь модуля теперь '
          '`github.com/arzab/gorock-catalog/{m}`. Модуль ещё не переведён на новое ядро: {note}.\n:::\n\n')

# module -> (repo, layer badge, where it lives in a GOROCK project, legacy note or None)
MODULES = {
    'rockengine': ('kit', 'Engine',
        'Сердце `cmd/main`: запускает и останавливает **apps** — HTTP, шину событий, планировщик. '
        'Сервисы — отдельный слой, его поднимает [rockservices](./rockservices) до движка. '
        'Контракт `App` — это интерфейс слоя Apps. → [Main](/architecture/main), [Apps](/architecture/apps)', None),
    'rockconfig': ('kit', 'Configs',
        'Загружает единый файл конфига в агрегированный `Configs` в `cmd/main`. '
        'Каждый слой объявляет свой `Configs`, rockconfig проверяет их все разом. '
        '→ [Конфигурация](/architecture/configs)', None),
    'rockservices': ('kit', 'Toolkit · Services',
        'Слой сервисов: `pkg/services/services.go` добавляет сервисы в каталог, `cmd/main` поднимает их '
        'до apps и останавливает после, любой слой получает сервис по имени. '
        '→ [Services](/architecture/services), [Main](/architecture/main)', None),
    'rockhandlers': ('kit', 'Realm · Delivery',
        'Форма обработки запроса в `internal/delivery`: цепочки шагов с `Next`, Locals, middleware, '
        'валидацией и видами ошибок — без привязки к HTTP или очереди. Транспорт (App) только заполняет '
        'Params и переводит результат в свой протокол. → [Delivery](/architecture/delivery)', None),
    'rockhttp': ('catalog', 'Engine · Apps',
        'HTTP App в `cmd/apps` на `net/http`: принимает маршруты rockhandlers из `internal/delivery/endpoints` '
        'как `Endpoint{Method, Path, Handler}`. Экспериментальный. '
        '→ [Apps](/architecture/apps), [Delivery](/architecture/delivery)', None),
    'rocklog': ('catalog', 'Infra',
        'Глобальный логгер проекта. `pkg/infra/logs` переводит настройки из файла в `rocklog.Init`, '
        'дальше все слои пишут через `rocklog`. → [Infra](/architecture/infra)',
        'логгер — инфраструктура, а не сервис; его API не меняется'),
    'rockfiber': ('catalog', 'Engine · Apps',
        'HTTP App на Fiber v2 со своей моделью endpoints (`fiber.Handler`, `DefaultHandler`, `NewError`). '
        '→ [Apps](/architecture/apps)',
        'для новых сервисов берите [rockhttp](./rockhttp) на [rockhandlers](/gorock-kit/rockhandlers)'),
    'rockbus': ('catalog', 'Engine · Apps',
        'Шина событий внутри процесса — App в `cmd/apps`. Подписки описываются в '
        '`internal/delivery/events`, публикация — через `rockbus.PublishAsync`. → [Apps](/architecture/apps)',
        'обработчики событий пока не на rockhandlers'),
    'rockcron': ('catalog', 'Engine · Apps',
        'Планировщик — App в `cmd/apps`. Задачи описываются в `internal/delivery/jobs`. → [Apps](/architecture/apps)',
        'задачи пока не на rockhandlers'),
    'rocktelebot': ('catalog', 'Engine · Apps',
        'Telegram-бот — App в `cmd/apps`. Команды и обработчики описываются в `internal/delivery`. '
        '→ [Apps](/architecture/apps)',
        'обработчики пока не на rockhandlers'),
    'rockbun': ('catalog', 'Toolkit · Services',
        'PostgreSQL через bun. Обычно — дочерний сервис внутри `pkg/services/repository`. '
        '→ [Services](/architecture/services)',
        'у сервиса нет `Name()`, поэтому в [rockservices](/gorock-kit/rockservices) его добавляют внутри своего сервиса '
        '(repository владеет rockbun), а не напрямую; в README ниже он ещё регистрируется как ресурс движка — '
        'этого механизма в kit v0.3.0 нет'),
    'rockredis': ('catalog', 'Toolkit · Services',
        'Redis для `pkg/services` (кэш, блокировки, pub/sub). → [Services](/architecture/services)',
        'у сервиса нет `Name()`, поэтому в [rockservices](/gorock-kit/rockservices) его добавляют внутри своего сервиса, '
        'а не напрямую; в README ниже он ещё регистрируется как ресурс движка — этого механизма в kit v0.3.0 нет'),
}


def page(module: str) -> tuple[str, str]:
    repo, layer, place, legacy = MODULES[module]
    checkout, section, url = REPOS[repo]
    src = os.path.join(checkout, module, 'README_RU.md')
    with open(src, encoding='utf-8') as f:
        text = f.read()

    lines = text.splitlines()
    assert lines[0].startswith('# '), f'{src}: first line must be the H1'
    title, body = lines[0], '\n'.join(lines[1:])

    # Links between modules point at their pages on the site.
    def link(m: re.Match) -> str:
        target = m.group(1)
        if target in MODULES:
            return f'](/{REPOS[MODULES[target][0]][1]}/{target})'
        return m.group(0)
    body = re.sub(r'\]\(\.\./(rock[a-z]+)/?\)', link, body)
    body = re.sub(r'\]\(https://github\.com/arzab/gorock-kit/tree/main/(rock[a-z]+)\)', link, body)

    header = (
        f'{title}\n\n'
        f'<!-- Сгенерировано docs/scripts/gen-kit-pages.py из {module}/README_RU.md — правьте README в репозитории модуля. -->\n\n'
        f'::: tip Место в GOROCK · {layer}\n{place}\n:::\n\n'
        + (LEGACY.format(m=module, note=legacy) if legacy else '') +
        f'```sh\ngo get {url.replace("https://", "")}/{module}@latest\n```\n\n'
        f'[Исходный код и README на GitHub]({url}/tree/main/{module})\n'
    )
    return section, header + body.rstrip() + '\n'


def main() -> None:
    for module in MODULES:
        section, text = page(module)
        out = os.path.join(ROOT, 'docs', section)
        os.makedirs(out, exist_ok=True)
        with open(os.path.join(out, f'{module}.md'), 'w', encoding='utf-8', newline='\n') as f:
            f.write(text)
        print(f'wrote {section}/{module}.md')


if __name__ == '__main__':
    main()
