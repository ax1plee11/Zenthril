#!/usr/bin/env python3
"""Zenthril Dissertation Generator - Creates professional Microsoft Word .docx"""

from docx import Document
from docx.shared import Pt, Cm, Inches, RGBColor
from docx.enum.text import WD_ALIGN_PARAGRAPH, WD_BREAK
from docx.enum.table import WD_TABLE_ALIGNMENT
import os

def set_margins(doc, left=Cm(30), right=Cm(10), top=Cm(20), bottom=Cm(20)):
    section = doc.sections[0]
    section.left_margin = left
    section.right_margin = right
    section.top_margin = top
    section.bottom_margin = bottom

def add_para(doc, text, size=Pt(14), bold=False, italic=False, align=None):
    p = doc.add_paragraph()
    run = p.add_run(text)
    run.font.name = "Times New Roman"
    run.font.size = size
    run.bold = bold
    run.italic = italic
    if align:
        p.alignment = align
    return p

def add_heading(doc, text, level=1):
    p = doc.add_paragraph()
    p.style = f"Heading {level}"
    run = p.add_run(text)
    run.font.name = "Times New Roman"
    run.font.size = Pt(14)
    run.bold = True
    return p

def add_table(doc, headers, rows):
    table = doc.add_table(rows=1, cols=len(headers))
    table.style = 'Table Grid'
    table.alignment = WD_TABLE_ALIGNMENT.CENTER
    hdr_cells = table.rows[0].cells
    for i, h in enumerate(headers):
        r = hdr_cells[i].paragraphs[0].add_run(h)
        r.font.name = "Times New Roman"
        r.font.size = Pt(12)
        r.bold = True
        hdr_cells[i].paragraphs[0].alignment = WD_ALIGN_PARAGRAPH.CENTER
    for row in rows:
        row_cells = table.add_row().cells
        for i, cell_text in enumerate(row):
            r = row_cells[i].paragraphs[0].add_run(str(cell_text))
            r.font.name = "Times New Roman"
            r.font.size = Pt(12)
    return table

doc = Document()
set_margins(doc)

# ===================== TITLE PAGE =====================
section = doc.sections[0]
section.different_first_page = True

t = doc.add_paragraph()
t.alignment = WD_ALIGN_PARAGRAPH.CENTER
add_para(t, "РАЗРАБОТКА АРХИТЕКТУРЫ ЛОКАЛЬНОГО STORAGE ENGINE С ЖУРНАЛИРОВАНИЕМ И КОНТРОЛЕМ ЦЕЛОСТНОСТИ ДАННЫХ НА ПРИМЕРЕ ZENTHRILDB", Pt(14), bold=True)

doc.add_paragraph()
doc.add_paragraph()

t2 = doc.add_paragraph()
t2.alignment = WD_ALIGN_PARAGRAPH.CENTER
add_para(t2, "ВЫПОЛНИЛ:", Pt(14))

add_table(doc, ["ФИО", "Студенческий номер", "Группа"], [["_____________________", "_____________________", "_____________________"]])

doc.add_paragraph()
t3 = doc.add_paragraph()
t3.alignment = WD_ALIGN_PARAGRAPH.CENTER
add_para(t3, "НАУЧНЫЙ РУКОВОДИТЕЛЬ:", Pt(14))
add_table(doc, ["ФИО", "Ученая степень"], [["_____________________", "_____________________"]])

doc.add_paragraph()
t4 = doc.add_paragraph()
t4.alignment = WD_ALIGN_PARAGRAPH.CENTER
add_para(t4, "Москва\n2026", Pt(14))

doc.add_page_break()

# ===================== АННОТАЦИЯ =====================
add_heading(doc, "АННОТАЦИЯ", level=1)
doc.add_paragraph()
add_para(doc, (
    "В диссертационной работе рассматривается архитектура локального storage engine ZenthrilDB — "
    "C++23 проекта для страничного локального хранилища с журналированием и контролем целостности. "
    "Проанализированы архитектурные решения, реализованные подсистемы и экспериментальные результаты. "
    "Описана бинарная файловая форма .zdb, реализованы и протестированы компоненты FileManager, "
    "BufferManager, MetadataManager, PageManager, FreePageManager, WAL, RecoveryEngine, "
    "UndoRedoRecoveryExecutor и LockManager. Проведено сравнение с аналогичными embedded storage "
    "engines: SQLite, RocksDB и PostgreSQL. Результаты анализа подтверждены модульными тестами C++ и Go. "
    "Диссертация включает экспериментальную оценку и рекомендации по дальнейшему развитию системы."
))
doc.add_paragraph()
add_para(doc, "Ключевые слова: storage engine, embedded database, C++23, WAL, CRC32, LRU buffer, recovery, deadlock detection")
doc.add_page_break()

# ===================== ABSTRACT =====================
add_heading(doc, "ABSTRACT", level=1)
doc.add_paragraph()
add_para(doc, (
    "This thesis presents the architecture of ZenthrilDB, a local-first storage engine implemented in C++23. "
    "The work analyzes the design and implementation of page-oriented storage with write-ahead logging, "
    "CRC32 integrity checking, LRU buffer management, metadata persistence, and local concurrency control. "
    "The current implementation status of each subsystem is documented against source code evidence. "
    "A comparative architectural analysis against SQLite, RocksDB, and PostgreSQL is provided. "
    "Experimental measurements confirm correctness properties. The dissertation concludes with identified "
    "limitations and future development directions including B+Tree and MVCC foundations."
))
doc.add_paragraph()
add_para(doc, "Keywords: storage engine, embedded database, C++23, WAL, CRC32, LRU buffer, recovery, deadlock detection")
doc.add_page_break()

# ===================== ОГЛАВЛЕНИЕ =====================
add_heading(doc, "ОГЛАВЛЕНИЕ", level=1)
doc.add_paragraph()
add_para(doc, "В Microsoft Word выберите все (Ctrl+A) и нажмите F9 для обновления оглавления.")
doc.add_page_break()

# ===================== СОКРАЩЕНИЯ =====================
add_heading(doc, "ПЕРЕЧЕНЬ СОКРАЩЕНИЙ И ОБОЗНАЧЕНИЙ", level=1)
doc.add_paragraph()
headers = ["Сокращение", "Расшифровка"]
rows = [
    ("API", "Application Programming Interface — интерфейс программирования приложений"),
    ("CRC", "Cyclic Redundancy Check — циклический избыточный код"),
    ("CPU", "Central Processing Unit"),
    ("DDL", "Data Definition Language — язык определения данных"),
    ("DH", "Diffie-Hellman — протокол обмена ключами"),
    ("E2EE", "End-to-End Encryption — сквозное шифрование"),
    ("FIFO", "First-In, First-Out"),
    ("HTTP", "HyperText Transfer Protocol"),
    ("JWT", "JSON Web Token"),
    ("LRU", "Least Recently Used — замещение наименее используемого"),
    ("MVCC", "Multi-Version Concurrency Control"),
    ("WAL", "Write-Ahead Log — журнал предзаписи"),
    ("LSN", "Log Sequence Number — номер последовательности журнала"),
    ("AES", "Advanced Encryption Standard"),
    ("SQL", "Structured Query Language"),
    ("X3DH", "Extended Triple Diffie-Hellman"),
)
add_table(doc, headers, rows)
doc.add_page_break()

# ===================== ВВЕДЕНИЕ =====================
add_heading(doc, "ВВЕДЕНИЕ", level=1)
doc.add_paragraph()
add_heading(doc, "Актуальность темы.", level=2)
doc.add_paragraph()
add_para(doc, (
    "Разработка надёжных систем хранения данных является одной из ключевых задач "
    "информационных технологий. Локальные (embedded) storage engine обеспечивают "
    "надёжное страничное хранилище с поддержкой журналирования, контроля целостности "
    "и восстановления после сбоев, что критически важно для local-first приложений. "
    "Анализ существующих решений выявляет необходимость исследования композиционных "
    "подходов к построению storage engine с явным управлением страницами, WAL и подсистемами блокировок."
))

add_heading(doc, "Объект и предмет исследования.", level=2)
doc.add_paragraph()
add_para(doc, (
    "Объект исследования — архитектура локального storage engine для страничного "
    "хранения данных с журналированием и контролем целостности.\n"
    "Предмет исследования — построение и реализация подсистем страничного хранилища, "
    "буферного менеджера, менеджера метаданных, системы свободных страниц, журнала "
    "предзаписи, менеджера транзакций, системы восстановления и менеджера блокировок "
    "на примере проекта ZenthrilDB."
))

add_heading(doc, "Цель и задачи.", level=2)
doc.add_paragraph()
add_para(doc, (
    "Цель работы — разработка и исследование архитектуры локального storage engine "
    "ZenthrilDB, обеспечивающего управляемое страничное хранение данных, контроль "
    "целостности, журналирование и расширяемую основу для восстановления после сбоев.\n\n"
    "Задачи исследования:\n"
    "1. Анализ аналогичных embedded storage engines (SQLite, RocksDB, PostgreSQL).\n"
    "2. Проектирование бинарного формата .zdb с фиксированными страницами 8192 байт.\n"
    "3. Реализация FileManager с CRC32 контролем целостности страниц.\n"
    "4. Реализация BufferManager с LRU политикой и RAII-защитой страниц.\n"
    "5. Реализация MetadataManager и PageManager для управления метаданами и страницами.\n"
    "6. Реализация WAL с LSN, CRC32 и проверкой целостности записей.\n"
    "7. Реализация транзакционного слоя с WAL-границами транзакций.\n"
    "8. Реализация RecoveryEngine с планированием восстановления.\n"
    "9. Реализация LockManager с deadlock detection.\n"
    "10. Проведение экспериментальной оценки и анализ ограничений."
))

add_heading(doc, "Методы исследования.", level=2)
doc.add_paragraph()
add_para(doc, (
    "В работе использованы методы анализа и синтеза архитектурных решений, "
    "компаративного анализа, экспериментальная оценка производительности, "
    "модульное тестирование и верификация бинарных форматов."
))

add_heading(doc, "Структура работы.", level=2)
doc.add_paragraph()
add_para(doc, (
    "Глава 1 — анализ предметной области и сравнение с аналогичными системами. "
    "Глава 2 — архитектура ZenthrilDB. Глава 3 — реализация подсистем. "
    "Глава 4 — надёжность и конкурентный доступ. Глава 5 — экспериментальная оценка. "
    "Глава 6 — перспективы развития и интеграция с Zenthril."
))
doc.add_page_break()

# ===================== ГЛАВА 1 =====================
add_heading(doc, "ГЛАВА 1. АНАЛИЗ ПРЕДМЕТНОЙ ОБЛАСТИ", level=1)

add_heading(doc, "1.1. Embedded storage engines: обзор аналогов", level=2)
doc.add_paragraph()
add_para(doc, (
    "Для анализа выбраны три эталонные системы: SQLite, RocksDB и PostgreSQL. "
    "Критерии сравнения: архитектура, формат хранения, страницы, WAL, recovery, "
    "concurrency, API, назначение и ограничения. Сравнение проводится без оценки "
    "«лучше/хуже» — только технический анализ подходов."
))

add_heading(doc, "1.2. Таблица сравнения архитектур", level=2)
doc.add_paragraph()

headers = ["Компонент", "ZenthrilDB", "SQLite", "RocksDB", "PostgreSQL"]
rows = [
    ["Архитектура", "Страничный, local-only", "Страничный, embedded", "LSM-tree, embedded", "Страничный, client-server"],
    ["Формат хранения", ".zdb (custom binary)", "SQLite format (custom binary)", "SSTable + WAL", "Page-based heap + WAL"],
    ["Размер страниц", "8192 байт (фиксированный)", "1–65536 байт (настраиваемый)", "страницы и sstable", "8–128 КБ (настраиваемый)"],
    ["WAL", "Да (.wal, LSN, CRC32)", "Да (журнал 提交)", "Да (WAL)", "Да (WAL)"],
    ["Recovery", "Частичный (анализ + план)", "Полный (rollback/rollforward)", "Полный", "Полный"],
    ["Concurrency", "LockManager с deadlock detection", "Однопоточный", "Lock-free", "MVCC + lightweight locks"],
    ["API", "C++23 API", "C API", "C++ API", "SQL + C API"],
    ["Назначение", "Local-first storage foundation", "Embedded SQL DB", "Embedded KV store", "Network SQL DB"],
    ["Ограничения", "Нет SQL, B+Tree, MVCC", "Однопоточный writes", "Нет SQL", "Требует сервер"],
]
add_table(doc, headers, rows)
doc.add_paragraph()
add_para(doc, (
    "Источник: [исходный код ZenthrilDB], документация SQLite [https://www.sqlite.org/arch.html], "
    "документация RocksDB [https://rocksdb.org/], документация PostgreSQL "
    "[https://www.postgresql.org/docs/current/architecture.html]"
), Pt(12))

add_heading(doc, "1.3. Структура .zdb файлового формата", level=2)
doc.add_paragraph()
add_para(doc, (
    "Файловая структура ZenthrilDB основана на фиксированных страницах 8192 байт. "
    "Каждая страница начинается с PageHeader (20 байт), остальная часть — payload. "
    "Файл .zdb содержит DatabaseHeader (72 байта) в начале, за которым следуют страницы."
))

doc.add_paragraph()
add_para(doc, "Рисунок 1.1 — Схема .zdb файлового формата", Pt(12), italic=True)
p = doc.add_paragraph()
set_font = lambda r: (setattr(r, 'font', None) or None)  # no-op
p.alignment = WD_ALIGN_PARAGRAPH.CENTER
run = p.add_run(
    "┌─────────────────────────────────────────────────┐\n" +
    "│ PAGE 0: DatabaseHeader (72 байта) + Header mirror│\n" +
    "├─────────────────────────────────────────────────┤\n" +
    "│ PAGE 1: DatabaseMetadata                        │\n" +
    "├─────────────────────────────────────────────────┤\n" +
    "│ PAGE 2: PageDirectory                           │\n" +
    "├─────────────────────────────────────────────────┤\n" +
    "│ PAGE 3: FreePageList                            │\n" +
    "├─────────────────────────────────────────────────┤\n" +
    "│ PAGE 4+: User data pages                        │\n" +
    "└─────────────────────────────────────────────────┘"
)
run.font.name = "Times New Roman"
run.font.size = Pt(10)
doc.add_page_break()

# ===================== ГЛАВА 2 =====================
add_heading(doc, "ГЛАВА 2. ПРОЕКТИРОВАНИЕ АРХИТЕКТУРЫ ZENTHRILDB", level=1)

add_heading(doc, "2.1. Общая архитектура ZenthrilDB", level=2)
doc.add_paragraph()
add_para(doc, "Архитектура ZenthrilDB состоит из следующих слоёв:\n"
    "1. Файловый слой (FileManager) — низкоуровневый доступ к .zdb и .wal.\n"
    "2. Буферный слой (BufferManager) — LRU кеш страниц с RAII.\n"
    "3. Слой страничного управления (PageManager) — lifecycle страниц.\n"
    "4. Слой метаданных (MetadataManager) — таблицы и каталоги.\n"
    "5. Слой транзакций (TransactionManager) — begin/commit/rollback.\n"
    "6. Слой WAL (LogManager) — журналирование с LSN и CRC32.\n"
    "7. Слой восстановления (RecoveryEngine) — анализ и планирование.\n"
    "8. Слой блокировок (LockManager) — конкурентный доступ и deadlock detection.")

add_heading(doc, "2.2. Архитектурные компоненты", level=2)

add_heading(doc, "2.2.1. DatabaseHeader и PageHeader", level=3)
doc.add_paragraph()
add_para(doc, (
    "DatabaseHeader (storage/DatabaseHeader.hpp) определяет магическое число, "
    "версию формата, размер страницы (8192 байт), общее количество страниц, "
    "метку времени создания, UUID базы данных и CRC32 контрольную сумму.\n\n"
    "PageHeader (storage/PageHeader.hpp) содержит pageId, тип страницы (Metadata, "
    "Table, Index, Free, Overflow), CRC32, usedBytes и version. "
    "Размер заголовка страницы фиксирован: 20 байт."
))

# Table 2.1
add_para(doc, "Таблица 2.1 — Поля DatabaseHeader", bold=True)
headers = ["Поле", "Тип", "Размер", "Описание"]
rows = [
    ["magic", "uint32", "4 байта", "Магическое число"],
    ["version", "uint32", "4 байта", "Версия формата БД"],
    ["pageSize", "uint32", "4 байта", "Размер страницы (8192)"],
    ["totalPages", "uint64", "8 байт", "Общее количество страниц"],
    ["creationTimestampUnixNs", "uint64", "8 байт", "Метка времени создания"],
    ["databaseUuid", "uint8[16]", "16 байт", "UUID базы данных"],
    ["checksum", "uint32", "4 байта", "CRC32 контрольная сумма"],
]
add_table(doc, headers, rows)

# Table 2.2
add_para(doc, "Таблица 2.2 — Поля PageHeader", bold=True)
headers = ["Поле", "Тип", "Размер", "Описание"]
rows = [
    ["pageId", "PageId (uint32)", "4 байта", "Идентификатор страницы"],
    ["pageType", "PageType (uint32)", "4 байта", "Тип: Metadata/Table/Index/Free/Overflow"],
    ["checksum", "uint32", "4 байта", "CRC32 контрольная сумма"],
    ["usedBytes", "uint32", "4 байта", "Используемые байты payload"],
    ["version", "uint32", "4 байта", "Версия страницы"],
]
add_table(doc, headers, rows)

add_para(doc, "Источник: [исходный код storage/DatabaseHeader.hpp/cpp, storage/PageHeader.hpp/cpp]", italic=True)

add_heading(doc, "2.2.2. FileManager", level=3)
doc.add_paragraph()
add_para(doc, (
    "FileManager (core/FileManager.hpp/cpp) обеспечивает доступ к файлу .zdb: "
    "createDatabase, openDatabase, readPage, writePage, flush, reservePageId. "
    "Thread-safe через recursive_mutex. При чтении вычисляется CRC32 и сравнивается "
    "с сохранённой — при несовпадении выбрасывается исключение."
))

add_heading(doc, "2.2.3. BufferManager и BufferPageGuard", level=3)
doc.add_paragraph()
add_para(doc, (
    "BufferManager (core/BufferManager.hpp/cpp) реализует LRU-кеш страниц с "
    "фиксированной ёмкостью. Операции: pin, pinGuard, markDirty, unpin, flushAll. "
    "При необходимости вытеснения ищет страницу с pinCount == 0, записывает если dirty."
))

add_heading(doc, "Алгоритм 7 — LRU eviction в BufferManager", level=3)
doc.add_paragraph()
add_para(doc, (
    "1. Если cache_.size() < capacityPages_ → выделить без вытеснения.\n"
    "2. Иначе пройти LRU-список от наименее используемой.\n"
    "3. Для каждой страницы проверить: pinCount == 0.\n"
    "4. Если dirty → fileManager_.writePage().\n"
    "5. Удалить страницу из LRU-списка и кэша.\n"
    "6. Если нет evictable → исключение."
))

add_heading(doc, "2.2.4. MetadataManager", level=3)
doc.add_paragraph()
add_para(doc, "MetadataManager управляет метаданными таблиц: создание, удаление, переименование, обновление. "
    "Метаданные сохраняются на PAGE 1, PageDirectory — на PAGE 2.")

add_heading(doc, "2.2.5. PageManager", level=3)
doc.add_paragraph()
add_para(doc, "PageManager управляет lifecycle страниц. Выделяет через FreePageManager, "
    "отклоняет запись в зарезервированные страницы, отслеживает dirty pages.")

add_heading(doc, "2.2.6. FreePageManager и FreePageList", level=3)
doc.add_paragraph()
add_para(doc, "FreePageManager и FreePageList управляют списком свободных страниц. "
    "Страницы переиспользуются перед расширением файла.")

add_heading(doc, "2.2.7. PageDirectory", level=3)
doc.add_paragraph()
add_para(doc, "PageDirectory сопоставляет page_id с метаданными: тип, owner_table, статус. "
    "Используется для отслеживания принадлежности страниц к таблицам.")

add_heading(doc, "2.2.8. WAL (Write-Ahead Log)", level=3)
doc.add_paragraph()
add_para(doc, "WAL состоит из: LSN, LogRecord (magic ZWAL, version, type, LSN, timestamp, payload_size, CRC32, payload), "
    "WriteAheadLog (файловый интерфейс).")

add_heading(doc, "2.2.9. LogManager", level=3)
doc.add_paragraph()
add_para(doc, "LogManager предоставляет потокобезопасный API: append, flush, checkpoint, replay, truncate, prepareRecovery.")

add_heading(doc, "2.2.10. Transaction layer", level=3)
doc.add_paragraph()
add_para(doc, "Transaction layer: TransactionId, Transaction, TransactionManager, TransactionLogAdapter. "
    "WAL типы: TransactionStarted, TransactionCommitted, TransactionRolledBack.")

add_heading(doc, "Алгоритм 10 — Жизненный цикл транзакции", level=3)
doc.add_paragraph()
add_para(doc, (
    "1. begin(): TransactionId, Transaction{Active}, WAL append(TransactionStarted).\n"
    "2. commit(id): WAL append(TransactionCommitted), markCommitted.\n"
    "3. rollback(id): WAL append(TransactionRolledBack), markRolledBack."
))

add_heading(doc, "2.2.11. Recovery", level=3)
doc.add_paragraph()
add_para(doc, "RecoveryEngine::analyze() находит последний checkpoint и строит RecoveryPlan. "
    "UndoRedoRecoveryExecutor применяет план с идемпотентностью по LSN.")

add_heading(doc, "2.2.12. LockManager", level=3)
doc.add_paragraph()
add_para(doc, "LockManager реализует: Shared/Exclusive modes, FIFO очередь с writer-preference, "
    "deadlock detection через WaitForGraph, timeout, lock upgrade/downgrade, RAII LockGuard.")

doc.add_heading("Алгоритм 13 — Обнаружение deadlock", level=3)
doc.add_paragraph()
add_para(doc, (
    "1. Построить adjacency list из wait-for graph.\n"
    "2. DFS с пометкой visited, recursion_stack, path.\n"
    "3. Если back edge → cycle найден.\n"
    "4. Вернуть участников цикла.\n"
    "5. SelectVictim: выбрать первую транзакцию (deterministic)."
))
doc.add_page_break()

# ===================== ГЛАВА 3 (continued)
add_heading(doc, "ГЛАВА 3. РЕАЛИЗОВАНИЕ ОСНОВНЫХ ПОДСИСТЕМ ZENTHRILDB", level=1)

add_heading(doc, "3.1. Бинарный формат .zdb", level=2)
doc.add_paragraph()
add_para(doc, (
    "Формат .zdb — бинарный файл с фиксированной структурой:\n\n"
    "1. PAGE 0 (offset 0): DatabaseHeader (72 байта) + Header mirror page.\n"
    "2. PAGE 1: DatabaseMetadata\n"
    "3. PAGE 2: PageDirectory\n"
    "4. PAGE 3: FreePageList\n"
    "5. PAGE 4+: пользовательские страницы.\n\n"
    "Используется little-endian для переносимости. Версионирование для расширения формата. "
    "CRC32 защищает страницы и WAL записи."
))

add_heading(doc, "Алгоритм 1 — Создание базы данных", level=3)
doc.add_paragraph()
add_para(doc, (
    "1. Открыть .zdb на запись.\n"
    "2. Создать DatabaseHeader с UUID, timestamp, magic, version, pageSize=8192.\n"
    "3. Вычислить CRC32 (checksum=0), записать header.\n"
    "4. Записать PAGE 0 (header mirror), PAGE 1 (DatabaseMetadata), PAGE 2 (PageDirectory), PAGE 3 (FreePageList). Flush."
))

add_heading(doc, "Алгоритм 4 — Запись страницы", level=3)
doc.add_paragraph()
add_para(doc, (
    "1. Обнулить checksum в header страницы.\n"
    "2. Сериализовать страницу, вычислить CRC32.\n"
    "3. Установить checksum.\n"
    "4. Seek на offset, записать 8192 байт.\n"
    "5. Если pageId+1 > totalPages → обновить header."
))

add_heading(doc, "3.2. WAL и LSN", level=2)
doc.add_paragraph()
add_para(doc, (
    "WAL реализован в wal/. Каждая запись: magic (ZWAL), version, type, LSN, timestamp, "
    "payload_size, CRC32, payload. Поддерживаемые типы: DatabaseCreated, DatabaseOpened, "
    "PageAllocated, PageFreed, PageWritten, MetadataUpdated, Checkpoint, RecoveryMarker, "
    "TransactionStarted, TransactionCommitted, TransactionRolledBack."
))

add_heading(doc, "Алгоритм 8 — WAL append", level=3)
doc.add_paragraph()
add_para(doc, (
    "1. LogRecord(lsn, type, payload): вычислить CRC32.\n"
    "2. Serialize: magic|version|type|lsn|timestamp|payload_size|crc32|payload.\n"
    "3. WriteAheadLog::append(): seek to end, write bytes. Return LSN."
))

add_heading(doc, "3.3. Транзакции", level=2)
doc.add_paragraph()
add_para(doc, "Transaction layer поддерживает: begin, commit, rollback с записями в WAL. "
    "Использует TransactionLogAdapter для записи событий транзакций.")

add_heading(doc, "3.4. Recovery", level=2)
doc.add_paragraph()
add_para(doc, "RecoveryEngine анализирует WAL, находит checkpoint, строит RecoveryPlan. "
    "UndoRedoRecoveryExecutor применяет план с идемпотентностью по LSN.")

doc.add_page_break()

# ===================== ГЛАВА 4 =================
add_heading(doc, "ГЛАВА 4. ИССЛЕДОВАНИЕ НАДЁЖНОСТИ, ЦЕЛОСТНОСТИ И КОНКУРЕНТНОГО ДОСТУПА", level=1)

add_heading(doc, "4.1. Контроль целостности (CRC32)", level=2)
doc.add_paragraph()
add_para(doc, "Каждая страница .zdb и каждая WAL-запись содержат CRC32. При записи "
    "контрольная сумма вычисляется с обнулённым полем. При чтении CRC32 вычисляется "
    "и сравнивается. Несовпадение → исключение.")

add_heading(doc, "4.2. WAL integrity", level=2)
doc.add_paragraph()
add_para(doc, "WAL проверяет: magic (ZWAL), version, payload_size vs фактический размер, "
    "CRC32, LSN монотонность. Искажения обнаруживаются и вызывают исключение.")

add_heading(doc, "4.3. Конкурентный доступ", level=2)
doc.add_paragraph()
add_para(doc, "LockManager: Shared/Exclusive, FIFO очередь с writer-preference, "
    "deadlock detection (WaitForGraph), timeout, lock upgrade/downgrade, RAII LockGuard.")

add_heading(doc, "4.4. Security hardening", level=2)
doc.add_paragraph()
add_para(doc, "EncryptionInterfaces существуют, но реализация не выполнена. "
    "Контроль целостности через CRC32 реализован.")
doc.add_page_break()

# ===================== ГЛАВА 5 =================
add_heading(doc, "ГЛАВА 5. ЭКСПЕРИМЕНТАЛЬНАЯ ОЦЕНКА", level=1)

add_heading(doc, "5.1. Результаты тестирования ZenthrilDB", level=2)
doc.add_paragraph()
add_para(doc, "Все 15 наборов тестов проходят успешно. Команда: zenthrildb_tests.exe → "
    "'All ZenthrilDB tests passed'.")

add_para(doc, "Таблица 5.1 — Результаты тестов ZenthrilDB", bold=True)
headers = ["Компонент", "Тест", "Статус", "Проверка"]
rows = [
    ["CRC32", "test_crc_manager", "PASS", "Корректность вычисления CRC32"],
    ["Binary IO", "test_binary_io", "PASS", "Сериализация little-endian"],
    ["Page Format", "test_page_format", "PASS", "Размер страницы, заголовок"],
    ["File Manager", "test_file_manager", "PASS", "CRUD страниц, CRC"],
    ["Buffer Manager", "test_buffer_manager", "PASS", "Pin/unpin, LRU, dirty"],
    ["Metadata", "test_metadata_serialization", "PASS", "Сериализация метаданных"],
    ["Free Page Manager", "test_free_page_manager", "PASS", "Allocate/free страниц"],
    ["Metadata Manager", "test_metadata_manager", "PASS", "Create/delete/rename"],
    ["Page Manager", "test_page_manager", "PASS", "Page lifecycle"],
    ["Storage Boundaries", "test_storage_boundaries", "PASS", "Границы страниц"],
    ["WAL", "test_wal", "PASS", "Append, replay, checkpoint, CRC, corruption"],
    ["Recovery", "test_recovery", "PASS", "Analysis, plan, execution"],
    ["Transaction", "test_transaction", "PASS", "Begin/commit/rollback"],
    ["Log Payload", "test_log_payload", "PASS", "Typed payload serialization"],
    ["Lock Manager", "test_lock_manager", "PASS", "Shared/exclusive, deadlock"],
]
add_table(doc, headers, rows)

add_heading(doc, "5.2. Результаты тестирования Zenthril", level=2)
doc.add_paragraph()
add_para(doc, "Go backend: все 29 пакетов PASS. "
    "Frontend: 174/176 тестов PASS. 2 теста WIP в pairwiseSession.security.test.ts "
    "(DH ratchet cross-side matching).")

add_heading(doc, "5.3. Corruption testing", level=2)
doc.add_paragraph()
add_para(doc, "В test_wal.cpp: проверка CRC corruption (XOR last byte), payload_size corruption. "
    "В test_file_manager.cpp: проверка CRC страниц.")

add_heading(doc, "5.4. Concurrency testing", level=2)
doc.add_paragraph()
add_para(doc, "test_lock_manager.cpp: concurrent shared locks (16 потоков), exclusive lock с timeout, "
    "lock upgrade, FIFO fairness. "
    "test_wal.cpp: concurrent WAL append (8 потоков). "
    "test_recovery.cpp: concurrent recovery (8 потоков).")

doc.add_page_break()

# ===================== ГЛАВА 6 =================
add_heading(doc, "ГЛАВА 6. ПЕРСПЕКТИВЫ РАЗВИТИЯ И ИНТЕГРАЦИЯ С ZENTHRIL", level=1)

add_heading(doc, "6.1. Текущие ограничения", level=2)
doc.add_paragraph()
add_para(doc, (
    "ZenthrilDB на текущем этапе — MVP foundation. Не реализовано:\n"
    "- MVCC Foundation (Planned: v0.10)\n"
    "- B+Tree Foundation (Planned: v0.11)\n"
    "- Catalog (Planned: v0.12)\n"
    "- Record Manager (Planned: v0.13)\n"
    "- Query Engine MVP (Planned: v0.14)\n"
    "- SQL Parser MVP (Planned: v0.15)\n"
    "- Optimizer Foundation (Planned: v0.16)\n"
    "- Local DB MVP v1.0 (Planned)\n"
    "- Encryption (out of scope до v1.0)"
))

add_heading(doc, "6.2. Интеграция с Zenthril", level=2)
doc.add_paragraph()
add_para(doc, (
    "Zenthril — messaging платформа на Go (backend) и Tauri/React/TypeScript (client). "
    "Интеграция ZenthrilDB с Zenthril не реализована. ZenthrilDB разворачивается как "
    "отдельный проект.\n\n"
    "Архитектурно предполагаемая модель интеграции:\n"
    "1. Go bindings для ZenthrilDB (CGO или FFI).\n"
    "2. Использование .zdb для локального хранения в Tauri client.\n"
    "3. Интеграция с DeviceKeyStore для локального E2EE ключевого хранилища."
))

doc.add_page_break()

# ===================== ЗАКЛЮЧЕНИЕ =================
add_heading(doc, "ЗАКЛЮЧЕНИЕ", level=1)
doc.add_paragraph()
add_para(doc, (
    "В работе выполнен аудит архитектуры ZenthrilDB. Реализованы и протестированы: "
    "страничное хранилище, BufferManager, MetadataManager, PageManager, FreePageManager, "
    "WAL с CRC32, транзакционный слой, RecoveryEngine, UndoRedoRecoveryExecutor и LockManager. "
    "Все unit-тесты C++ и Go проходят успешно. "
    "Интеграция ZenthrilDB с Zenthril не реализована — отдельные проекты. "
    "ZenthrilDB находится на этапе MVP foundation и не является production-ready DBMS."
))

add_heading(doc, "Вклад в науку и практическая значимость", level=2)
doc.add_paragraph()
add_para(doc, "Научная новизна: анализ композиционной архитектуры embedded storage engine "
    "на основе фиксированных страниц, WAL с CRC32, LRU буферного менеджера с RAII и lock manager с deadlock detection. "
    "Практическая значимость: storage engine как foundation для local-first приложений.")
doc.add_page_break()

# ===================== СПИСОК ИСТОЧНИКОВ ===================
add_heading(doc, "СПИСОК ИСПОЛЬЗОВАННЫХ ИСТОЧНИКОВ", level=1)
doc.add_paragraph()

ref_list = [
    ("SQLite Documentation", "https://www.sqlite.org/docs.html"),
    ("SQLite Architecture", "https://www.sqlite.org/arch.html"),
    ("SQLite File Format", "https://www.sqlite.org/fileformat.html"),
    ("SQLite WAL", "https://www.sqlite.org/wal.html"),
    ("RocksDB Documentation", "https://rocksdb.org/"),
    ("RocksDB Architecture", "https://github.com/facebook/rocksdb/wiki"),
    ("PostgreSQL Documentation", "https://www.postgresql.org/docs/"),
    ("PostgreSQL Storage Architecture", "https://www.postgresql.org/docs/current/storage.html"),
    ("PostgreSQL WAL Internals", "https://www.postgresql.org/docs/current/wal-internals.html"),
    ("PostgreSQL Concurrency Control", "https://www.postgresql.org/docs/current/mvcc-intro.html"),
    ("PostgreSQL Lock Management", "https://www.postgresql.org/docs/current/lock-management.html"),
    ("PostgreSQL Page Layout", "https://www.postgresql.org/docs/current/storage-page-layout.html"),
    ("PostgreSQL Transaction Isolation", "https://www.postgresql.org/docs/current/transaction-iso.html"),
    ("CRC-32", "https://en.wikipedia.org/wiki/Cyclic_redundancy_check"),
    ("Curve25519/X25519", "https://en.wikipedia.org/wiki/Curve25519"),
    ("Diffie-Hellman key exchange", "https://en.wikipedia.org/wiki/Diffie%E2%80%93Hellman_key_exchange"),
    ("ZenthrilDB source code", "[исходный код]"),
    ("Zenthril source code", "[исходный код]"),
    ("ZenthrilDB docs", "[docs/architecture.md]"),
    ("ZenthrilDB WAL_SPEC", "[docs/WAL_SPEC.md]"),
    ("ZenthrilDB RECOVERY_SPEC", "[docs/RECOVERY_SPEC.md]"),
    ("ZenthrilDB TRANSACTION_SPEC", "[docs/TRANSACTION_SPEC.md]"),
    ("ZenthrilDB LOCK_SPEC", "[docs/LOCK_SPEC.md]"),
    ("ZenthrilDB ROADMAP", "[docs/ROADMAP.md]"),
    ("Zenthril SECURITY.md", "[SECURITY.md]"),
    ("Zenthril THREAT_MODEL.md", "[THREAT_MODEL.md]"),
]

for i, (title, url) in enumerate(ref_list, 1):
    add_para(doc, f"[{i}] {title}. URL: {url}", Pt(12))

doc.add_page_break()

# ===================== ПРИЛОЖЕНИЯ =====================
add_heading(doc, "ПРИЛОЖЕНИЕ А. Структура проекта", level=1)
doc.add_paragraph()
add_para(doc, (
    "zenthrildb-push/\n"
    "├── CMakeLists.txt\n"
    "├── main.cpp\n"
    "├── core/ (CRCManager, BufferManager, FileManager, FreePageManager, MetadataManager, PageManager, EncryptionInterfaces)\n"
    "├── storage/ (DatabaseHeader, DatabaseMetadata, PageDirectory, Page, PageHeader, FreePageList, TableMetadata, BinaryIO, BinaryFormat)\n"
    "├── wal/ (WriteAheadLog, LogManager, LogRecord, LogPayload, LogSequenceNumber, RecoveryContext)\n"
    "├── transaction/ (Transaction, TransactionId, TransactionContext, TransactionLogAdapter, TransactionManager)\n"
    "├── recovery/ (RecoveryEngine, RecoveryExecutor, RecoveryPlan, CheckpointService)\n"
    "├── lock/ (LockManager, LockTypes, WaitForGraph)\n"
    "├── tests/ (15 наборов тестов)\n"
    "├── build/ (Windows сборка, .exe)\n"
    "└── docs/ (архитектура, спецификации, roadmap)\n"
))

doc.add_heading(doc, "ПРИЛОЖЕНИЕ Б. Основные структуры данных", level=1)
doc.add_paragraph()
add_para(doc, (
    "Определены в core/Types.hpp:\n"
    "- PageId: uint32_t\n"
    "- PageSize: uint32_t (8192)\n"
    "- Byte: uint8_t\n"
    "- kDatabaseMagic: 0x5A44424C\n"
    "- kCurrentDatabaseVersion: 1\n"
    "- kFirstUserPageId: 4\n"
    "- LogSequenceNumber: uint64_t обёртка"
))

doc.add_heading(doc, "ПРИЛОЖЕНИЕ В. Результаты тестирования", level=1)
doc.add_paragraph()
add_para(doc, (
    "ZenthrilDB: 15 наборов тестов — ВСЕ PASS. Команда выводит: 'All ZenthrilDB tests passed'\n"
    "Zenthril Go backend: 29 пакетов — ВСЕ PASS (go test ./...)\n"
    "Zenthril Frontend: 176 тестов — 174 PASS, 2 FAIL (WIP DH ratchet tests)"
))

doc.add_heading(doc, "ПРИЛОЖЕНИЕ Г. Конфигурация стенда", level=1)
doc.add_paragraph()
add_para(doc, (
    "OS: Windows 10/11\n"
    "C++ компилятор: MSVC (C++23)\n"
    "Сборка: CMake 3.20+\n"
    "Go: 1.23+\n"
    "Node.js: 20+\n"
    "Docker: 29.3.1\n"
    "Тесты: zenthrildb_tests.exe, go test, vitest"
))

doc.add_heading(doc, "ПРИЛОЖЕНИЕ Д. Дополнительные диаграммы", level=1)
doc.add_paragraph()
add_para(doc, "WAL record format:\n"
    "Offset 0: magic(4) | version(4) | type(4) | lsn(8) | timestamp(8) | payload_size(4) | crc32(4) | payload(N)")

doc.add_heading(doc, "ПРИЛОЖЕНИЕ Е. Таблица прослеживаемости", level=1)
doc.add_paragraph()
headers = ["Claim", "Source", "Code", "Test", "Section"]
rows = [
    ["CRC32 реализован", "docs/architecture.md", "core/CRCManager", "test_crc_manager.cpp", "2.2.14, 4.1"],
    ["FileManager читает/пишет страницы", "docs/architecture.md", "core/FileManager", "test_file_manager.cpp", "2.2.2, 3.2"],
    ["BufferManager с LRU", "docs/architecture.md", "core/BufferManager", "test_buffer_manager.cpp", "2.2.3, 3.1"],
    ["RAII BufferPageGuard", "docs/architecture.md", "core/BufferManager.hpp", "test_buffer_manager.cpp", "2.2.3"],
    ["MetadataManager управляет таблицами", "docs/architecture.md", "core/MetadataManager", "test_metadata_manager.cpp", "2.2.4, 3.3"],
    ["FreePageManager переиспользует страницы", "docs/architecture.md", "core/FreePageManager", "test_free_page_manager.cpp", "2.2.6"],
    ["PageManager управляет lifecycle", "docs/architecture.md", "core/PageManager", "test_page_manager.cpp", "2.2.5, 3.3"],
    ["WAL с LSN, CRC32, versioned records", "docs/WAL_SPEC.md", "wal/LogRecord", "test_wal.cpp", "2.2.8, 3.2, 4.2"],
    ["LogManager: append, replay, checkpoint", "docs/architecture.md", "wal/LogManager", "test_wal.cpp", "2.2.9"],
    ["Transaction: begin/commit/rollback", "docs/TRANSACTION_SPEC.md", "transaction/TransactionManager", "test_transaction.cpp", "2.2.10, 3.3"],
    ["RecoveryEngine анализирует WAL", "docs/RECOVERY_SPEC.md", "recovery/RecoveryEngine", "test_recovery.cpp", "2.2.11, 3.4"],
    ["UndoRedoRecoveryExecutor с идемпотентностью", "docs/architecture.md", "recovery/RecoveryExecutor", "test_recovery.cpp", "2.2.12, 3.4"],
    ["LockManager с deadlock detection", "docs/LOCK_SPEC.md", "lock/LockManager", "test_lock_manager.cpp", "2.2.13, 4.3, 3.5"],
    ["WaitForGraph для deadlock detection", "docs/LOCK_SPEC.md", "lock/WaitForGraph", "test_lock_manager.cpp", "2.2.13"],
    ["Encryption НЕ реализован", "docs/architecture.md", "core/EncryptionInterfaces.hpp", "Нет тестов", "2.2.14, 4.4, 6.1"],
    ["MVCC Foundation — Planned", "docs/ROADMAP.md", "Нет исходников", "Нет тестов", "6.1"],
    ["B+Tree Foundation — Planned", "docs/ROADMAP.md", "Нет исходников", "Нет тестов", "6.1"],
    ["ZenthrilDB ≠ Zenthril интеграция", "README.md", "Нет интеграции", "Нет тестов", "6.2"],
    ["Zenthril backend: Go, тесты PASS", "README.md", "backend/", "go test ./...", "6.2"],
    ["Frontend: 174/176 PASS", "README.md", "client/src/features/e2ee/", "npm test", "5.2"],
    ["2 WIP теста DH ratchet", "pairwiseSession.security.test.ts", "pairwiseSession.ts", "test", "5.2"],
    ["Полный crash recovery НЕ реализован", "docs/RECOVERY_SPEC.md", "recovery/RecoveryExecutor.cpp", "test_recovery.cpp", "2.2.12, 6.1"],
    ["SQL НЕ реализован", "docs/ROADMAP.md", "Нет SQL parser", "Нет тестов", "6.1"],
]
add_table(doc, headers, rows)

doc.add_heading(doc, "ПРИЛОЖЕНИЕ Ж. Валидация", level(1))
doc.add_paragraph()
headers = ["Утверждение", "Статус", "Доказательство"]
rows = [
    ["CRC32 реализован", "PASS", "test_crc_manager.cpp"],
    ["FileManager с CRC", "PASS", "test_file_manager.cpp"],
    ["BufferManager с LRU", "PASS", "test_buffer_manager.cpp"],
    ["WAL с LSN и CRC32", "PASS", "test_wal.cpp"],
    ["RecoveryEngine анализирует WAL", "PASS", "test_recovery.cpp"],
    ["UndoRedoRecoveryExecutor", "PASS", "test_recovery.cpp"],
    ["LockManager с deadlock detection", "PASS", "test_lock_manager.cpp"],
    ["Transaction layer", "PASS", "test_transaction.cpp"],
    ["Encryption реализован", "НЕТ", "Только интерфейсы"],
    ["SQL реализован", "НЕТ", "Planned: v0.15"],
    ["B+Tree реализован", "НЕТ", "Planned: v0.11"],
    ["MVCC реализован", "НЕТ", "Planned: v0.10"],
    ["ZenthrilDB интегрирован с Zenthril", "НЕТ", "Отдельные проекты"],
    ["Полный crash recovery", "НЕТ", "Только анализ + планирование"],
    ["All ZenthrilDB tests pass", "PASS", "zenthrildb_tests.exe"],
    ["All Go backend tests pass", "PASS", "go test ./..."],
    ["174/176 frontend tests pass", "PASS", "npm test"],
]
add_table(doc, headers, rows)

# Save
output_path = os.path.join(os.getcwd(), "Zenthril_Dissertation.docx")
doc.save(output_path)

# Validation report
val_path = os.path.join(os.getcwd(), "Zenthril_Dissertation_Validation.md")
with open(val_path, "w", encoding="utf-8") as f:
    f.write("# Dissertation Validation Report\n\n")
    f.write("| Claim | Source | Status | Evidence |\n")
    f.write("|-------|--------|--------|----------|\n")
    f.write("| CRC32 реализован | core/CRCManager.hpp/cpp | ✓ PASS | test_crc_manager.cpp |\n")
    f.write("| FileManager читает/пишет страницы | core/FileManager.hpp/cpp | ✓ PASS | test_file_manager.cpp |\n")
    f.write("| BufferManager с LRU | core/BufferManager.hpp/cpp | ✓ PASS | test_buffer_manager.cpp |\n")
    f.write("| WAL с LSN и CRC32 | wal/LogRecord.hpp/cpp | ✓ PASS | test_wal.cpp |\n")
    f.write("| RecoveryEngine | recovery/RecoveryEngine.hpp/cpp | ✓ PASS | test_recovery.cpp |\n")
    f.write("| UndoRedoRecoveryExecutor | recovery/RecoveryExecutor.hpp/cpp | ✓ PASS | test_recovery.cpp |\n")
    f.write("| LockManager с deadlock detection | lock/LockManager.hpp/cpp | ✓ PASS | test_lock_manager.cpp |\n")
    f.write("| Transaction layer | transaction/TransactionManager.hpp/cpp | ✓ PASS | test_transaction.cpp |\n")
    f.write("| Encryption НЕ реализован | core/EncryptionInterfaces.hpp | ✗ PLANNED | Только интерфейсы |\n")
    f.write("| SQL НЕ реализован | ROADMAP.md | ✗ PLANNED | v0.15+ |\n")
    f.write("| ZenthrilDB ≠ Zenthril интеграция | отдельные проекты | ✗ NONE | Нет интеграции |\n")
    f.write("| All ZenthrilDB tests pass | zenthrildb_tests.exe | ✓ PASS | 'All ZenthrilDB tests passed' |\n")
    f.write("| All Go backend tests pass | go test ./... | ✓ PASS | 29 packages ok |\n")
    f.write("| 174/176 frontend tests pass | npm test | ✓ PASS | 2 WIP DH ratchet tests |\n")

print(f"Dissertation saved to: {output_path}")
print(f"Validation saved to: {val_path}")
print(f"File exists: {os.path.exists(output_path)}")
