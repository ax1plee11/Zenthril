import sys
from pathlib import Path

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))

import pytest  # noqa: E402

from etl.models import (  # noqa: E402
    CanonicalRecord,
    ETLPipeline,
    KnowledgeRepository,
    RecordStatus,
    SourceManifest,
)
