/*
 * Copyright (c) 2023 - for information on the respective copyright owner
 * see the NOTICE file and/or the repository https://github.com/carbynestack/klyshko.
 *
 * SPDX-License-Identifier: Apache-2.0
 */
#include "Tools/ezOptionParser.h"

#include "Math/Setup.h"
#include "Math/field_types.h"
#include "Math/gfp.hpp"
#include "Math/gf2n.h"
#include "Machines/SPDZ.hpp"
#include "Protocols/CowGearOptions.h"
#include "Protocols/CowGearShare.h"
#include "Protocols/CowGearPrep.hpp"
#include "Tools/Buffer.h"
#include <fstream>
#include <assert.h>
#include <boost/filesystem.hpp>

/**
 * Outputs the specified number of tuples of the given type into a file in the given directory.
 *
 * @param preprocessing The preprocessing implementation.
 * @param tuple_type The tuple type to be generated.
 * @param tuple_count The number of tuples to be generated.
 * @param tuple Container for storing elements of a tuple.
 * @param working_dir The directory within which the tuple file is created.
 * @param player_id The zero-based number of the local player.
 */
template <class T, std::size_t ELEMENTS>
void generate_tuples(Preprocessing<T> &preprocessing, Dtype tuple_type, int tuple_count, array<T, ELEMENTS> &tuple, string &working_dir, int player_id)
{

    // Open file to which tuples are written (equals "<DIR>/<TYPE=Triples|Squares|Bits|Inverses>-<FIELD=p|2>-P<PLAYER_ID>"")
    char filename[2048];
    sprintf(filename, (working_dir + "%s-%s-P%d").c_str(), DataPositions::dtype_names[tuple_type],
            (T::type_short()).c_str(), player_id);
    ofstream fout;
    fout.open(filename, ios::binary | ios::out);
    assert(fout.is_open());
    file_signature<T>().output(fout);

    // Generate and output the tuples
    std::cout << "Generating " << tuple_count << " tuples of type " << DataPositions::dtype_names[tuple_type] << std::endl;
    for (int i = 1; i <= tuple_count; ++i)
    {
        preprocessing.get(tuple_type, tuple.data());
        for (const T &ele : tuple)
        {
            ele.output(fout, false);
        }
    }

    std::cout << "Wrote " << tuple_count << " tuples of type " << DataPositions::dtype_names[tuple_type] << " to " << filename << std::endl;
    fout.close();
}

template <class T>
void generate_tuples(int player_id, int number_of_players, int port, int tuple_count, string tuple_type, string playerfile)
{
    // Create working directory, if it doesn't exist yet
    std::string working_dir = get_prep_sub_dir<T>(PREP_DIR, number_of_players);
    boost::filesystem::path path(working_dir);
    if (!(boost::filesystem::exists(path)))
    {
        if (boost::filesystem::create_directory(path))
            std::cout << "Non-existing working directory " << path << " created" << std::endl;
    }

    // Init networking
    Names N;
    N.init(player_id, port, playerfile, number_of_players);

    PlainPlayer P(N);

    T::clear::template write_setup<T>(number_of_players);

    // Initialize MAC key, if not existing
    typename T::mac_key_type mac_key;
    T::read_or_generate_mac_key(PREP_DIR, P, mac_key);
    write_mac_key(PREP_DIR, player_id, number_of_players, mac_key);

    // Required for keeping track of preprocessing material
    DataPositions usage(P.num_players());

    // Configure the player to use given MAC key
    typename T::MAC_Check output(mac_key);
    output.setup(P);

    // Initialize preprocessing
    CowGearPrep<T> preprocessing(0, usage);
    SubProcessor<T> processor(output, preprocessing, P);

    // Generate requested tuples
    if (tuple_type == "bits")
    {
        array<T, 1> bit;
        generate_tuples<T, 1>(preprocessing, DATA_BIT, tuple_count, bit, working_dir, player_id);
    }
    else if (tuple_type == "squares")
    {
        array<T, 2> square;
        generate_tuples<T, 2>(preprocessing, DATA_SQUARE, tuple_count, square, working_dir, player_id);
    }
    else if (tuple_type == "inverses")
    {
        array<T, 2> inverse;
        generate_tuples<T, 2>(preprocessing, DATA_INVERSE, tuple_count, inverse, working_dir, player_id);
    }
    else if (tuple_type == "triples")
    {
        array<T, 3> triple;
        generate_tuples<T, 3>(preprocessing, DATA_TRIPLE, tuple_count, triple, working_dir, player_id);
    }
    else
    {
        std::cerr << "Tuple type not supported: " << tuple_type << std::endl;
        exit(EXIT_FAILURE);
    }

    // Perform the MAC check
    output.Check(P);
}

int main(int argc, const char **argv)
{
    // Define and parse command line parameters
    ez::ezOptionParser opt;
    CowGearOptions(opt, argc, argv);
    opt.add(
        "",                   // Default value
        1,                    // Required?
        1,                    // Number of values expected
        0,                    // Delimiter, if expecting multiple args
        "Number of parties",  // Help description
        "-N",                 // Short name
        "--number-of-parties" // Long name
    );
    opt.add(
        "",                                                  // Default value
        1,                                                   // Required?
        1,                                                   // Number of values expected
        0,                                                   // Delimiter, if expecting multiple args
        "This player's number, starting with 0 (required).", // Help description
        "-p",                                                // Short name
        "--player"                                           // Long name
    );
    opt.add(
        "players",                                              // Default value
        0,                                                      // Required?
        1,                                                      // Number of values expected
        0,                                                      // Delimiter, if expecting multiple args
        "Playerfile containing host:port information per line", // Help description
        "-pf",                                                  // Short name
        "--playerfile"                                          // Long name
    );
    opt.add(
        "",                                        // Default value
        1,                                         // Required?
        1,                                         // Number of values expected
        0,                                         // Delimiter, if expecting multiple args
        "The field type to use. One of gfp, gf2n", // Help description
        "-ft",                                     // Short name
        "--field-type"                             // Long name
    );
    opt.add(
        "",                                                                    // Default value
        1,                                                                     // Required?
        1,                                                                     // Number of values expected
        0,                                                                     // Delimiter, if expecting multiple args
        "Tuple type to be generated. One of bits, inverses, squares, triples", // Help description
        "-tt",                                                                 // Short name
        "--tuple-type"                                                         // Long name
    );
    opt.add(
        "5000",                              // Default value
        0,                                   // Required?
        1,                                   // Number of values expected
        0,                                   // Delimiter, if expecting multiple args
        "Local port number (default: 5000)", // Help description
        "-P",                                // Short name
        "--port"                             // Long name
    );
    opt.add(
        "100000",                                         // Default value
        0,                                                // Required?
        1,                                                // Number of values expected
        0,                                                // Delimiter, if expecting multiple args
        "Number of tuples to generate (default: 100000)", // Help description
        "-tc",                                            // Short name
        "--tuple-count"                                   // Long name
    );
    opt.parse(argc, argv);
    if (!opt.isSet("-N") || !opt.isSet("-p") || !opt.isSet("-ft") || !opt.isSet("-tt"))
    {
        string usage;
        opt.getUsage(usage);
        cout << usage;
        exit(0);
    }

    int player_id, number_of_players, port, tuple_count;
    string field_type, tuple_type, playerfile;
    opt.get("-p")->getInt(player_id);
    opt.get("-N")->getInt(number_of_players);
    opt.get("-P")->getInt(port);
    opt.get("-ft")->getString(field_type);
    opt.get("-tt")->getString(tuple_type);
    opt.get("-tc")->getInt(tuple_count);
    opt.get("-pf")->getString(playerfile);

    if (mkdir_p(PREP_DIR) == -1)
    {
        throw runtime_error(
            (string) "cannot use " + PREP_DIR + " (set another PREP_DIR in CONFIG when building if needed)");
    }

    if (field_type == "gfp")
    {

        // Compute number of 64-bit words needed
        const int prime_length = 128;
        const int n_limbs = (prime_length + 63) / 64;

        // Define share type
        // (first argument is counter. convention is 0 for online, 1 for offline phase)
        typedef CowGearShare<gfp_<1, n_limbs>> T;

        // Initialize field
        T::clear::init_default(prime_length);

        generate_tuples<T>(player_id, number_of_players, port, tuple_count, tuple_type, playerfile);
    }
    else if (field_type == "gf2n")
    {
        // Define share type
        typedef CowGearShare<gf2n_short> T;

        // Initialize field
        gf2n_short::init_field(40);

        generate_tuples<T>(player_id, number_of_players, port, tuple_count, tuple_type, playerfile);
    } else {
        std::cerr << "Field type not supported: " << field_type << std::endl;
        exit(EXIT_FAILURE);
    }

}
