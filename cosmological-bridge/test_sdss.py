from astroquery.sdss import SDSS
query = "SELECT TOP 10 specObjID, ra, dec, z FROM SpecObj WHERE class='GALAXY'"
try:
    res = SDSS.query_sql(query)
    print(res)
except Exception as e:
    print(e)
