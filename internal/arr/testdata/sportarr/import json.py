import json


if __name__ == "__main__":
    data = None
    with open("v3_series.json") as f:
        data = json.load(f)

    new_data = []
    for series in data:
        new_series = {
            "id": series["id"],
            "monitored": series["monitored"],
            "seasons": [],
            "statistics": {
                "seasonCount": series["statistics"]["seasonCount"],
                "episodeFileCount": series["statistics"]["episodeFileCount"],
                "episodeCount": series["statistics"]["episodeCount"],
                "totalEpisodeCount": series["statistics"]["totalEpisodeCount"],
                "sizeOnDisk": series["statistics"]["sizeOnDisk"],
                "percentOfEpisodes": series["statistics"]["percentOfEpisodes"],
            },
        }

        for season in series["seasons"]:
            new_season = {
                "monitored": season["monitored"],
                "statistics": {
                    "episodeFileCount": season["statistics"]["episodeFileCount"],
                    "episodeCount": season["statistics"]["episodeCount"],
                    "totalEpisodeCount": season["statistics"]["totalEpisodeCount"],
                    "sizeOnDisk": season["statistics"]["sizeOnDisk"],
                    "percentOfEpisodes": season["statistics"]["percentOfEpisodes"],
                },
            }
            if "episodeFileCount" in season.keys():
                new_season["episodeFileCount"] = season["episodeFileCount"]
            if "episodeCount" in season.keys():
                new_season["episodeCount"] = season["episodeCount"]
            if "totalEpisodeCount" in season.keys():
                new_season["totalEpisodeCount"] = season["totalEpisodeCount"]
            new_series["seasons"].append(new_season)
        new_data.append(new_series)
    with open("v3_series_new.json", "w") as f:
        json.dump(new_data, f, indent=4)