import threading
from concurrent.futures import ThreadPoolExecutor

import pytest

import py_excelize as pex

from conftest import EXPECTED_CONTROLS


def test_shared_workbook_across_threads(controls_path):
    with pex.open(controls_path) as wb:
        def work(_):
            return len(wb["Sheet 1"].controls)

        with ThreadPoolExecutor(8) as pool:
            assert set(pool.map(work, range(200))) == {len(EXPECTED_CONTROLS)}


def test_one_workbook_per_thread(cells_path):
    results = []

    def work():
        with pex.open(cells_path) as wb:
            results.append(wb["Sheet1"]["A2"])

    threads = [threading.Thread(target=work) for _ in range(16)]
    for t in threads:
        t.start()
    for t in threads:
        t.join()
    assert results == ["apple"] * 16


def test_close_races_with_reads(cells_path):
    wb = pex.open(cells_path)
    errors = []

    def reader():
        for _ in range(50):
            try:
                wb["Sheet1"].rows
            except pex.WorkbookClosedError:
                return
            except Exception as exc:  # pragma: no cover
                errors.append(exc)
                return

    threads = [threading.Thread(target=reader) for _ in range(4)]
    for t in threads:
        t.start()
    wb.close()
    for t in threads:
        t.join()
    assert errors == []


def test_many_opens_in_pool(cells_path):
    def work(i):
        with pex.open(cells_path) as wb:
            return wb.sheetnames[i % 3]

    with ThreadPoolExecutor(8) as pool:
        out = list(pool.map(work, range(60)))
    assert out == [["Sheet1", "Sheet 2", "Ünï Çødé"][i % 3] for i in range(60)]


@pytest.mark.parametrize("n", [1, 10])
def test_load_from_threads(cells_path, n):
    data = cells_path.read_bytes()
    with ThreadPoolExecutor(n) as pool:
        names = list(pool.map(lambda _: pex.load(data).sheetnames, range(n * 3)))
    assert all(x == names[0] for x in names)
