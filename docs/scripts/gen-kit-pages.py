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
    'rockengine': ('kit', 'Main · движок App',
        'Движок, которым Main запускает и останавливает **App**: по порядку, с перезапуском упавших и остановкой '
        'по одному в обратном порядке. Сервисы — не его забота, их открывает и закрывает [rockservices](./rockservices). '
        '→ [Main](/guide/main), [Как писать App](/guide/apps)', None),
    'rockconfig': ('kit', 'Config',
        'Загружает единый файл настроек в `Configs` Main и проверяет его целиком при старте. '
        '→ [Как писать Config](/guide/configs)', None),
    'rockservices': ('kit', 'Services',
        'Каталог сервисов: `pkg/services/services.go` объявляет типизированные ключи и регистрирует сервисы, Main открывает их до App '
        'и закрывает после, любой код получает сервис по ключу: `services.Orders.Get()`. → [Как писать Service](/guide/services)', None),
    'rockhandlers': ('kit', 'Handlers',
        'Обработчики в `internal/handlers`: цепочки шагов, параметры и результат, ошибки с видом, общие шаги, '
        'middleware — без привязки к технологии. App заполняет параметры и переводит итог на свой язык. '
        '→ [Как писать Handler](/guide/handlers)', None),
    'rockhttp': ('catalog', 'App',
        'HTTP App на `net/http`: в `cmd/apps/http.go` вы привязываете обработчики к методам и путям '
        '(`Endpoint{Method, Path, Handler}`), он заполняет параметры и переводит итог в HTTP. Экспериментальный. '
        '→ [Как писать App](/guide/apps)', None),
    'rocklog': ('catalog', 'Services · логгер',
        'Логгер проекта на `log/slog`. Настраивается сервисом-логгером, который регистрируется в каталоге первым; '
        'дальше все пишут через `rocklog.Info`, `rocklog.FromContext(ctx)`. → [Как писать Service](/guide/services#логгер-тоже-сервис)',
        'API не меняется; в README ниже логгер настраивается из `pkg/infra` — теперь это `Init` сервиса-логгера'),
    'rockfiber': ('catalog', 'App',
        'HTTP App на Fiber v2 со своей моделью endpoints (`fiber.Handler`, `DefaultHandler`, `NewError`). '
        '→ [Как писать App](/guide/apps)',
        'его endpoints — не обработчики rockhandlers; для новых проектов берите [rockhttp](./rockhttp)'),
    'rockbus': ('catalog', 'App',
        'Шина событий внутри процесса — App в `cmd/apps`. Публикация — `rockbus.PublishAsync`. → [Как писать App](/guide/apps)',
        'его подписчики — не обработчики rockhandlers'),
    'rockcron': ('catalog', 'App',
        'Планировщик — App в `cmd/apps`, запускает задачи по расписанию. → [Как писать App](/guide/apps)',
        'его задачи — не обработчики rockhandlers'),
    'rocktelebot': ('catalog', 'App',
        'Telegram-бот — App в `cmd/apps`. → [Как писать App](/guide/apps)',
        'его обработчики — не обработчики rockhandlers'),
    'rockbun': ('catalog', 'Services',
        'Клиент PostgreSQL на bun. Обычно — дочерний сервис внутри вашего, например `pkg/services/orders`. '
        'Его `Service` — это `Init` + `Stop`, поэтому его можно и зарегистрировать в каталоге напрямую: '
        '`rockservices.Add(c, Postgres, rockbun.NewService, cfg.Postgres)`. → [Как писать Service](/guide/services)',
        'в README ниже он регистрируется как ресурс движка (`RegisterResource`) — такого механизма в ядре больше нет; '
        'регистрируйте его через [rockservices](/gorock-kit/rockservices) или держите внутри своего сервиса'),
    'rockredis': ('catalog', 'Services',
        'Клиент Redis: строки, хэши, списки, pub/sub. Дочерний сервис внутри вашего или самостоятельный сервис в каталоге: '
        '`rockservices.Add(c, Cache, rockredis.NewService, cfg.Cache)`. → [Как писать Service](/guide/services)',
        'в README ниже он регистрируется как ресурс движка (`RegisterResource`) — такого механизма в ядре больше нет; '
        'регистрируйте его через [rockservices](/gorock-kit/rockservices)'),
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
    body = re.sub(r'\]\(\.\./(rock[a-z]+)(?:/README(?:_RU)?\.md)?/?\)', link, body)
    body = re.sub(r'\]\(https://github\.com/arzab/gorock-kit/tree/main/(rock[a-z]+)\)', link, body)
    for g_repo, g_module, g_src, g_name, _ in GUIDES:  # links to guides inside the module
        if g_module == module:
            body = body.replace(f']({g_src})', f'](./{g_name})')

    header = (
        f'{title}\n\n'
        f'<!-- Сгенерировано docs/scripts/gen-kit-pages.py из {module}/README_RU.md — правьте README в репозитории модуля. -->\n\n'
        f'::: tip Место в GOROCK · {layer}\n{place}\n:::\n\n'
        + (LEGACY.format(m=module, note=legacy) if legacy else '') +
        f'```sh\ngo get {url.replace("https://", "")}/{module}@latest\n```\n\n'
        f'[Исходный код и README на GitHub]({url}/tree/main/{module})\n'
    )
    return section, header + body.rstrip() + '\n'


# Extra guides: (repo, module, source file, page name, unused)
GUIDES = [
    ('kit', 'rockhandlers', 'TRANSPORT_RU.md', 'rockhandlers-transport', None),
]


def guide(repo: str, module: str, src_name: str) -> tuple[str, str]:
    checkout, section, url = REPOS[repo]
    src = os.path.join(checkout, module, src_name)
    with open(src, encoding='utf-8') as f:
        text = f.read()
    text = re.sub(r'\]\(\.\./(rock[a-z]+)(?:/README(?:_RU)?\.md)?/?\)',
                  lambda m: f'](/{REPOS[MODULES[m.group(1)][0]][1]}/{m.group(1)})' if m.group(1) in MODULES else m.group(0), text)
    text = text.replace('https://github.com/arzab/gorock-catalog/tree/main/rockhttp', '/catalog/rockhttp')
    text = text.replace('https://arzab.github.io/gorock/guide/apps', '/guide/apps')
    lines = text.splitlines()
    header = (
        f'{lines[0]}\n\n'
        f'<!-- Сгенерировано docs/scripts/gen-kit-pages.py из {module}/{src_name} — правьте файл в репозитории модуля. -->\n\n'
        f'[Исходный файл на GitHub]({url}/blob/main/{module}/{src_name})\n'
    )
    return section, header + '\n'.join(lines[1:]).rstrip() + '\n'


def main() -> None:
    for repo, module, src_name, name, _ in GUIDES:
        section, text = guide(repo, module, src_name)
        with open(os.path.join(ROOT, 'docs', section, f'{name}.md'), 'w', encoding='utf-8', newline='\n') as f:
            f.write(text)
        print(f'wrote {section}/{name}.md')
    for module in MODULES:
        section, text = page(module)
        out = os.path.join(ROOT, 'docs', section)
        os.makedirs(out, exist_ok=True)
        with open(os.path.join(out, f'{module}.md'), 'w', encoding='utf-8', newline='\n') as f:
            f.write(text)
        print(f'wrote {section}/{module}.md')


if __name__ == '__main__':
    main()
